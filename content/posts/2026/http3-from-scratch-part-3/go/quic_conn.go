package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

// Stream, ReceiveStream, SendStream, Conn, and Listener are the exact interfaces
// our HTTP/3 Server and Client from Part 2 use!
type Stream interface {
	io.ReadWriteCloser
	StreamID() int64
}

type ReceiveStream interface {
	io.Reader
	StreamID() int64
}

type SendStream interface {
	io.WriteCloser
	StreamID() int64
}

type Conn interface {
	AcceptStream(ctx context.Context) (Stream, error)
	OpenStreamSync(ctx context.Context) (Stream, error)
	AcceptUniStream(ctx context.Context) (ReceiveStream, error)
	OpenUniStreamSync(ctx context.Context) (SendStream, error)
	RemoteAddr() net.Addr
	Context() context.Context
	CloseWithError(code uint64, desc string) error
}

type Listener interface {
	Accept(ctx context.Context) (Conn, error)
	Addr() net.Addr
	Close() error
}

// toyQUICStream implements an independent byte stream over UDP with offset reassembly!
type toyQUICStream struct {
	id          uint64
	conn        *toyQUICConn
	mu          sync.Mutex
	cond        *sync.Cond
	chunks      map[uint64][]byte // out-of-order and in-order chunks keyed by stream byte offset
	readOffset  uint64
	writeOffset uint64
	finReceived bool
	finOffset   uint64
	closed      bool
}

func newToyQUICStream(id uint64, conn *toyQUICConn) *toyQUICStream {
	s := &toyQUICStream{
		id:     id,
		conn:   conn,
		chunks: make(map[uint64][]byte),
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *toyQUICStream) StreamID() int64 { return int64(s.id) }

// pushFrame inserts an incoming STREAM frame at its exact byte offset and wakes any blocked Reader.
// Notice how packet reordering on UDP or loss on another stream never blocks this stream if its
// own contiguous bytes have arrived!
func (s *toyQUICStream) pushFrame(offset uint64, data []byte, fin bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(data) > 0 {
		copied := make([]byte, len(data))
		copy(copied, data)
		s.chunks[offset] = copied
	}
	if fin {
		s.finReceived = true
		s.finOffset = offset + uint64(len(data))
	}
	s.cond.Broadcast()
}

func (s *toyQUICStream) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if chunk, ok := s.chunks[s.readOffset]; ok {
			n := copy(p, chunk)
			if n == len(chunk) {
				delete(s.chunks, s.readOffset)
			} else {
				s.chunks[s.readOffset+uint64(n)] = chunk[n:]
				delete(s.chunks, s.readOffset)
			}
			s.readOffset += uint64(n)
			return n, nil
		}
		if s.finReceived && s.readOffset >= s.finOffset {
			return 0, io.EOF
		}
		if s.closed {
			return 0, io.ErrClosedPipe
		}
		s.cond.Wait()
	}
}

// Write chops data into <=1000-byte STREAM frames so each QUIC packet fits comfortably
// inside a single standard 1200+ byte UDP datagram without IP fragmentation.
func (s *toyQUICStream) Write(p []byte) (int, error) {
	total := 0
	const maxChunk = 1000
	for len(p) > 0 {
		chunk := p
		if len(chunk) > maxChunk {
			chunk = p[:maxChunk]
		}
		p = p[len(chunk):]

		s.mu.Lock()
		offset := s.writeOffset
		s.writeOffset += uint64(len(chunk))
		s.mu.Unlock()

		err := s.conn.sendFrame(QUICFrame{
			Type:     QUICFrameStreamBase,
			StreamID: s.id,
			Offset:   offset,
			Data:     chunk,
		})
		if err != nil {
			return total, err
		}
		total += len(chunk)
	}
	return total, nil
}

// Close sends an empty STREAM frame with the FIN bit set, signaling end-of-stream to the peer.
func (s *toyQUICStream) Close() error {
	s.mu.Lock()
	offset := s.writeOffset
	s.mu.Unlock()

	return s.conn.sendFrame(QUICFrame{
		Type:     QUICFrameStreamBase,
		StreamID: s.id,
		Offset:   offset,
		Fin:      true,
	})
}

type toyQUICConn struct {
	udpConn      net.PacketConn
	remoteAddr   net.Addr
	localConnID  uint64
	remoteConnID uint64
	isServer     bool
	pktNum       atomic.Uint32

	mu         sync.Mutex
	streams    map[uint64]*toyQUICStream
	nextBidiSeq uint64
	nextUniSeq  uint64

	acceptBidiCh chan Stream
	acceptUniCh  chan ReceiveStream
	handshakeCh  chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
}

func newToyQUICConn(udpConn net.PacketConn, remoteAddr net.Addr, localCID, remoteCID uint64, isServer bool) *toyQUICConn {
	ctx, cancel := context.WithCancel(context.Background())
	return &toyQUICConn{
		udpConn:      udpConn,
		remoteAddr:   remoteAddr,
		localConnID:  localCID,
		remoteConnID: remoteCID,
		isServer:     isServer,
		streams:      make(map[uint64]*toyQUICStream),
		acceptBidiCh: make(chan Stream, 32),
		acceptUniCh:  make(chan ReceiveStream, 32),
		handshakeCh:  make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (c *toyQUICConn) sendFrame(f QUICFrame) error {
	pkt := Packet{
		IsLongHeader: false, // 1-RTT Short Header packet
		DestConnID:   c.remoteConnID,
		PacketNumber: c.pktNum.Add(1),
		Frames:       []QUICFrame{f},
	}
	_, err := c.udpConn.WriteTo(EncodePacket(pkt), c.remoteAddr)
	return err
}

func (c *toyQUICConn) handlePacket(pkt Packet) {
	var shouldAck bool
	for _, f := range pkt.Frames {
		switch f.Type {
		case QUICFrameCrypto:
			shouldAck = true
			select {
			case <-c.handshakeCh:
			default:
				close(c.handshakeCh)
			}

		case QUICFrameStreamBase:
			shouldAck = true
			c.mu.Lock()
			st, exists := c.streams[f.StreamID]
			if !exists {
				st = newToyQUICStream(f.StreamID, c)
				c.streams[f.StreamID] = st
				isPeerInitiated := (f.StreamID&StreamIDServerInitiated != 0) != c.isServer
				if isPeerInitiated {
					if f.StreamID&StreamIDUnidirectional != 0 {
						c.acceptUniCh <- st
					} else {
						c.acceptBidiCh <- st
					}
				}
			}
			c.mu.Unlock()
			st.pushFrame(f.Offset, f.Data, f.Fin)

		case QUICFrameConnectionClose:
			c.cancel()
		}
	}

	if shouldAck {
		_ = c.sendFrame(QUICFrame{
			Type:         QUICFrameAck,
			LargestAcked: uint64(pkt.PacketNumber),
		})
	}
}

func (c *toyQUICConn) OpenStreamSync(ctx context.Context) (Stream, error) {
	c.mu.Lock()
	seq := c.nextBidiSeq
	c.nextBidiSeq++
	// Stream ID: sequence in upper bits, bit 1 = 0 (bidirectional), bit 0 = initiator
	streamID := seq << 2
	if c.isServer {
		streamID |= StreamIDServerInitiated
	}
	st := newToyQUICStream(streamID, c)
	c.streams[streamID] = st
	c.mu.Unlock()
	return st, nil
}

func (c *toyQUICConn) OpenUniStreamSync(ctx context.Context) (SendStream, error) {
	c.mu.Lock()
	seq := c.nextUniSeq
	c.nextUniSeq++
	// Stream ID: sequence in upper bits, bit 1 = 1 (unidirectional), bit 0 = initiator
	streamID := (seq << 2) | StreamIDUnidirectional
	if c.isServer {
		streamID |= StreamIDServerInitiated
	}
	st := newToyQUICStream(streamID, c)
	c.streams[streamID] = st
	c.mu.Unlock()
	return st, nil
}

func (c *toyQUICConn) AcceptStream(ctx context.Context) (Stream, error) {
	select {
	case st := <-c.acceptBidiCh:
		return st, nil
	case <-c.ctx.Done():
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *toyQUICConn) AcceptUniStream(ctx context.Context) (ReceiveStream, error) {
	select {
	case st := <-c.acceptUniCh:
		return st, nil
	case <-c.ctx.Done():
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *toyQUICConn) RemoteAddr() net.Addr     { return c.remoteAddr }
func (c *toyQUICConn) Context() context.Context { return c.ctx }

func (c *toyQUICConn) CloseWithError(code uint64, desc string) error {
	_ = c.sendFrame(QUICFrame{
		Type:         QUICFrameConnectionClose,
		ErrorCode:    code,
		ReasonPhrase: desc,
	})
	c.cancel()
	if !c.isServer {
		return c.udpConn.Close()
	}
	return nil
}

// toyQUICListener multiplexes multiple QUIC connections over a single UDP socket
// using their 64-bit Destination Connection ID!
type toyQUICListener struct {
	udpConn  net.PacketConn
	mu       sync.Mutex
	conns    map[uint64]*toyQUICConn
	acceptCh chan Conn
	closed   chan struct{}
}

func ListenToyQUIC(addr string, _ *tls.Config) (Listener, error) {
	udpConn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}
	l := &toyQUICListener{
		udpConn:  udpConn,
		conns:    make(map[uint64]*toyQUICConn),
		acceptCh: make(chan Conn, 16),
		closed:   make(chan struct{}),
	}
	go l.readLoop()
	return l, nil
}

func (l *toyQUICListener) readLoop() {
	buf := make([]byte, 65535)
	for {
		n, raddr, err := l.udpConn.ReadFrom(buf)
		if err != nil {
			return
		}
		pkt, err := DecodePacket(buf[:n])
		if err != nil {
			continue
		}

		l.mu.Lock()
		conn, exists := l.conns[pkt.DestConnID]
		if !exists && pkt.IsLongHeader && pkt.LongType == PacketTypeInitial {
			serverCID := randomConnID()
			conn = newToyQUICConn(l.udpConn, raddr, serverCID, pkt.SrcConnID, true)
			l.conns[serverCID] = conn
			// Also map the client's initial destination CID so 0-RTT or early packets route properly
			l.conns[pkt.DestConnID] = conn

			// Respond with a Long Header Handshake packet carrying a CRYPTO frame and ACK
			respPkt := Packet{
				IsLongHeader: true,
				LongType:     PacketTypeHandshake,
				Version:      QUICVersion1,
				DestConnID:   pkt.SrcConnID,
				SrcConnID:    serverCID,
				PacketNumber: conn.pktNum.Add(1),
				Frames: []QUICFrame{
					{Type: QUICFrameAck, LargestAcked: uint64(pkt.PacketNumber)},
					{Type: QUICFrameCrypto, CryptoData: []byte("SERVER_HELLO_ALPN_H3")},
				},
			}
			_, _ = l.udpConn.WriteTo(EncodePacket(respPkt), raddr)
			l.acceptCh <- conn
		}
		l.mu.Unlock()

		if conn != nil {
			conn.handlePacket(pkt)
		}
	}
}

func (l *toyQUICListener) Accept(ctx context.Context) (Conn, error) {
	select {
	case c := <-l.acceptCh:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *toyQUICListener) Addr() net.Addr { return l.udpConn.LocalAddr() }
func (l *toyQUICListener) Close() error {
	close(l.closed)
	return l.udpConn.Close()
}

// DialToyQUIC establishes a QUIC connection over UDP by sending an Initial Long Header packet
// with a CRYPTO frame and waiting for the server's Handshake response.
func DialToyQUIC(ctx context.Context, addr string, _ *tls.Config) (Conn, error) {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	clientCID := randomConnID()
	initialDestCID := randomConnID()
	conn := newToyQUICConn(udpConn, raddr, clientCID, initialDestCID, false)

	go func() {
		buf := make([]byte, 65535)
		for {
			n, _, err := udpConn.ReadFrom(buf)
			if err != nil {
				conn.cancel()
				return
			}
			pkt, err := DecodePacket(buf[:n])
			if err != nil {
				continue
			}
			if pkt.IsLongHeader && pkt.SrcConnID != 0 {
				conn.remoteConnID = pkt.SrcConnID
			}
			conn.handlePacket(pkt)
		}
	}()

	// Send Initial Packet with ClientHello CRYPTO frame
	initPkt := Packet{
		IsLongHeader: true,
		LongType:     PacketTypeInitial,
		Version:      QUICVersion1,
		DestConnID:   initialDestCID,
		SrcConnID:    clientCID,
		PacketNumber: conn.pktNum.Add(1),
		Frames: []QUICFrame{
			{Type: QUICFrameCrypto, CryptoData: []byte("CLIENT_HELLO_ALPN_H3")},
		},
	}
	if _, err := udpConn.WriteTo(EncodePacket(initPkt), raddr); err != nil {
		udpConn.Close()
		return nil, err
	}

	select {
	case <-conn.handshakeCh:
		return conn, nil
	case <-ctx.Done():
		udpConn.Close()
		return nil, errors.New("quic handshake timed out")
	}
}

func randomConnID() uint64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return binary.BigEndian.Uint64(b[:])
}

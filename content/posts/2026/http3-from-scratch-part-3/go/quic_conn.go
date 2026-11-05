package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

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

type quicStream struct {
	id          uint64
	conn        *quicConn
	mu          sync.Mutex
	cond        *sync.Cond
	chunks      map[uint64][]byte
	readOffset  uint64
	writeOffset uint64
	finReceived bool
	finOffset   uint64
	closed      bool
}

func newQUICStream(id uint64, conn *quicConn) *quicStream {
	s := &quicStream{
		id:     id,
		conn:   conn,
		chunks: make(map[uint64][]byte),
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *quicStream) StreamID() int64 { return int64(s.id) }

func (s *quicStream) pushFrame(offset uint64, data []byte, fin bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(data) > 0 {
		s.chunks[offset] = bytes.Clone(data)
	}
	if fin {
		s.finReceived = true
		s.finOffset = offset + uint64(len(data))
	}
	s.cond.Broadcast()
}

func (s *quicStream) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if chunk, ok := s.chunks[s.readOffset]; ok {
			n := copy(p, chunk)
			delete(s.chunks, s.readOffset)
			if n < len(chunk) {
				s.chunks[s.readOffset+uint64(n)] = chunk[n:]
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

func (s *quicStream) Write(p []byte) (int, error) {
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

		err := s.conn.send1RTTFrame(QUICFrame{
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

func (s *quicStream) Close() error {
	s.mu.Lock()
	offset := s.writeOffset
	s.mu.Unlock()

	return s.conn.send1RTTFrame(QUICFrame{
		Type:     QUICFrameStreamBase,
		StreamID: s.id,
		Offset:   offset,
		Fin:      true,
	})
}

type quicConn struct {
	udpConn      net.PacketConn
	remoteAddr   net.Addr
	localConnID  []byte
	remoteConnID []byte
	isServer     bool
	pktNum       atomic.Uint32

	tlsConn           *tls.QUICConn
	readKeys          [4]*packetKeys
	writeKeys         [4]*packetKeys
	expectedReadPN    [4]uint64
	cryptoWriteOffset [4]uint64
	cryptoReadOffset  [4]uint64
	cryptoReadChunks  [4]map[uint64][]byte

	mu          sync.Mutex
	streams     map[uint64]*quicStream
	nextBidiSeq uint64
	nextUniSeq  uint64

	acceptBidiCh chan Stream
	acceptUniCh  chan ReceiveStream
	handshakeCh  chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
}

func newQUICConn(udpConn net.PacketConn, remoteAddr net.Addr, localCID, remoteCID []byte, isServer bool, tlsConn *tls.QUICConn) *quicConn {
	ctx, cancel := context.WithCancel(context.Background())
	c := &quicConn{
		udpConn:      udpConn,
		remoteAddr:   remoteAddr,
		localConnID:  bytes.Clone(localCID),
		remoteConnID: bytes.Clone(remoteCID),
		isServer:     isServer,
		tlsConn:      tlsConn,
		streams:      make(map[uint64]*quicStream),
		acceptBidiCh: make(chan Stream, 32),
		acceptUniCh:  make(chan ReceiveStream, 32),
		handshakeCh:  make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
	}
	for i := range c.cryptoReadChunks {
		c.cryptoReadChunks[i] = make(map[uint64][]byte)
	}
	return c
}

func levelIndex(l tls.QUICEncryptionLevel) int {
	switch l {
	case tls.QUICEncryptionLevelInitial:
		return 0
	case tls.QUICEncryptionLevelHandshake:
		return 1
	case tls.QUICEncryptionLevelApplication:
		return 2
	default:
		return 3
	}
}

func (c *quicConn) getReadKeys(isLong bool, longType uint8) (*packetKeys, uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !isLong {
		return c.readKeys[2], c.expectedReadPN[2]
	}
	if longType == PacketTypeInitial {
		return c.readKeys[0], c.expectedReadPN[0]
	}
	if longType == PacketTypeHandshake {
		return c.readKeys[1], c.expectedReadPN[1]
	}
	return nil, 0
}

func (c *quicConn) sendPacketAtLevel(level int, frames []QUICFrame) error {
	c.mu.Lock()
	keys := c.writeKeys[level]
	dcid := bytes.Clone(c.remoteConnID)
	scid := bytes.Clone(c.localConnID)
	c.mu.Unlock()
	if keys == nil {
		return errors.New("missing write keys for level")
	}

	pkt := Packet{
		Version:      QUICVersion1,
		DestConnID:   dcid,
		SrcConnID:    scid,
		PacketNumber: c.pktNum.Add(1),
		Frames:       frames,
	}
	minSize := 0
	switch level {
	case 0:
		pkt.IsLongHeader = true
		pkt.LongType = PacketTypeInitial
		if !c.isServer {
			minSize = 1200 // RFC 9000 Section 14.1 minimum client Initial size
		}
	case 1:
		pkt.IsLongHeader = true
		pkt.LongType = PacketTypeHandshake
	case 2:
		pkt.IsLongHeader = false
	}

	raw, err := EncodeProtectedPacket(pkt, keys, minSize)
	if err != nil {
		return err
	}
	_, err = c.udpConn.WriteTo(raw, c.remoteAddr)
	return err
}

func (c *quicConn) send1RTTFrame(f QUICFrame) error {
	return c.sendPacketAtLevel(2, []QUICFrame{f})
}

func (c *quicConn) drainTLSEvents() error {
	for {
		ev := c.tlsConn.NextEvent()
		switch ev.Kind {
		case tls.QUICNoEvent:
			return nil

		case tls.QUICSetReadSecret:
			keys, err := derivePacketKeys(ev.Suite, ev.Data)
			if err != nil {
				return err
			}
			c.mu.Lock()
			c.readKeys[levelIndex(ev.Level)] = keys
			c.mu.Unlock()

		case tls.QUICSetWriteSecret:
			keys, err := derivePacketKeys(ev.Suite, ev.Data)
			if err != nil {
				return err
			}
			c.mu.Lock()
			c.writeKeys[levelIndex(ev.Level)] = keys
			c.mu.Unlock()

		case tls.QUICWriteData:
			idx := levelIndex(ev.Level)
			data := ev.Data
			const maxChunk = 900
			for len(data) > 0 {
				chunk := data
				if len(chunk) > maxChunk {
					chunk = data[:maxChunk]
				}
				data = data[len(chunk):]

				c.mu.Lock()
				offset := c.cryptoWriteOffset[idx]
				c.cryptoWriteOffset[idx] += uint64(len(chunk))
				c.mu.Unlock()

				err := c.sendPacketAtLevel(idx, []QUICFrame{{
					Type:         QUICFrameCrypto,
					CryptoOffset: offset,
					CryptoData:   bytes.Clone(chunk),
				}})
				if err != nil {
					return err
				}
			}

		case tls.QUICHandshakeDone:
			if c.isServer {
				_ = c.send1RTTFrame(QUICFrame{Type: QUICFrameHandshakeDone})
			}
			select {
			case <-c.handshakeCh:
			default:
				close(c.handshakeCh)
			}
		}
	}
}

func (c *quicConn) getOrCreateStream(streamID uint64) *quicStream {
	c.mu.Lock()
	defer c.mu.Unlock()

	st, exists := c.streams[streamID]
	if !exists {
		st = newQUICStream(streamID, c)
		c.streams[streamID] = st
		isPeerInitiated := (streamID&StreamIDServerInitiated != 0) != c.isServer
		if isPeerInitiated {
			if streamID&StreamIDUnidirectional != 0 {
				c.acceptUniCh <- st
			} else {
				c.acceptBidiCh <- st
			}
		}
	}
	return st
}

func (c *quicConn) readCrypto(level int, offset uint64, data []byte) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cryptoReadChunks[level][offset] = bytes.Clone(data)
	var contiguous []byte
	for {
		chunk, ok := c.cryptoReadChunks[level][c.cryptoReadOffset[level]]
		if !ok {
			break
		}
		delete(c.cryptoReadChunks[level], c.cryptoReadOffset[level])
		c.cryptoReadOffset[level] += uint64(len(chunk))
		contiguous = append(contiguous, chunk...)
	}
	return contiguous
}

func (c *quicConn) handlePacket(pkt Packet) {
	level := 2
	tlsLevel := tls.QUICEncryptionLevelApplication
	if pkt.IsLongHeader {
		if len(pkt.SrcConnID) > 0 {
			c.mu.Lock()
			c.remoteConnID = bytes.Clone(pkt.SrcConnID)
			c.mu.Unlock()
		}
		if pkt.LongType == PacketTypeInitial {
			level = 0
			tlsLevel = tls.QUICEncryptionLevelInitial
		} else if pkt.LongType == PacketTypeHandshake {
			level = 1
			tlsLevel = tls.QUICEncryptionLevelHandshake
		}
	}

	c.mu.Lock()
	if uint64(pkt.PacketNumber)+1 > c.expectedReadPN[level] {
		c.expectedReadPN[level] = uint64(pkt.PacketNumber) + 1
	}
	c.mu.Unlock()

	var shouldAck bool
	for _, f := range pkt.Frames {
		switch f.Type {
		case QUICFramePing, QUICFrameHandshakeDone, QUICFrameNewConnectionID:
			shouldAck = true

		case QUICFramePathChallenge:
			shouldAck = true
			_ = c.sendPacketAtLevel(level, []QUICFrame{{
				Type: QUICFramePathResponse,
				Data: f.Data,
			}})

		case QUICFrameCrypto:
			shouldAck = true
			if data := c.readCrypto(level, f.CryptoOffset, f.CryptoData); len(data) > 0 && c.tlsConn != nil {
				_ = c.tlsConn.HandleData(tlsLevel, data)
				_ = c.drainTLSEvents()
			}

		case QUICFrameStreamBase:
			shouldAck = true
			c.getOrCreateStream(f.StreamID).pushFrame(f.Offset, f.Data, f.Fin)

		case QUICFrameConnectionClose:
			c.cancel()
			c.mu.Lock()
			for _, st := range c.streams {
				st.mu.Lock()
				st.closed = true
				st.cond.Broadcast()
				st.mu.Unlock()
			}
			c.mu.Unlock()
		}
	}

	if shouldAck {
		_ = c.sendPacketAtLevel(level, []QUICFrame{{
			Type:         QUICFrameAck,
			LargestAcked: uint64(pkt.PacketNumber),
		}})
	}
}

func (c *quicConn) OpenStreamSync(ctx context.Context) (Stream, error) {
	c.mu.Lock()
	seq := c.nextBidiSeq
	c.nextBidiSeq++
	streamID := seq << 2
	if c.isServer {
		streamID |= StreamIDServerInitiated
	}
	st := newQUICStream(streamID, c)
	c.streams[streamID] = st
	c.mu.Unlock()
	return st, nil
}

func (c *quicConn) OpenUniStreamSync(ctx context.Context) (SendStream, error) {
	c.mu.Lock()
	seq := c.nextUniSeq
	c.nextUniSeq++
	streamID := (seq << 2) | StreamIDUnidirectional
	if c.isServer {
		streamID |= StreamIDServerInitiated
	}
	st := newQUICStream(streamID, c)
	c.streams[streamID] = st
	c.mu.Unlock()
	return st, nil
}

func (c *quicConn) AcceptStream(ctx context.Context) (Stream, error) {
	select {
	case st := <-c.acceptBidiCh:
		return st, nil
	case <-c.ctx.Done():
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *quicConn) AcceptUniStream(ctx context.Context) (ReceiveStream, error) {
	select {
	case st := <-c.acceptUniCh:
		return st, nil
	case <-c.ctx.Done():
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *quicConn) RemoteAddr() net.Addr     { return c.remoteAddr }
func (c *quicConn) Context() context.Context { return c.ctx }

func (c *quicConn) CloseWithError(code uint64, desc string) error {
	_ = c.send1RTTFrame(QUICFrame{
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

type quicListener struct {
	udpConn   net.PacketConn
	tlsConfig *tls.Config
	mu        sync.Mutex
	conns     map[string]*quicConn
	acceptCh  chan Conn
	closed    chan struct{}
}

func ListenQUIC(addr string, tlsConf *tls.Config) (Listener, error) {
	udpConn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}
	l := &quicListener{
		udpConn:   udpConn,
		tlsConfig: tlsConf,
		conns:     make(map[string]*quicConn),
		acceptCh:  make(chan Conn, 16),
		closed:    make(chan struct{}),
	}
	go l.readLoop()
	return l, nil
}

func (l *quicListener) readLoop() {
	buf := make([]byte, 65535)
	for {
		n, raddr, err := l.udpConn.ReadFrom(buf)
		if err != nil {
			return
		}
		datagram := buf[:n]

		l.mu.Lock()
		conn := l.conns[raddr.String()]
		if conn == nil && len(datagram) > 6 && datagram[0]&0x80 != 0 {
			dcidLen := int(datagram[5])
			if len(datagram) >= 6+dcidLen+1 {
				initialDCID := bytes.Clone(datagram[6 : 6+dcidLen])
				scidLen := int(datagram[6+dcidLen])
				var clientSCID []byte
				if len(datagram) >= 7+dcidLen+scidLen {
					clientSCID = bytes.Clone(datagram[7+dcidLen : 7+dcidLen+scidLen])
				}
				serverCID := randomConnID()
				srvTLS := l.tlsConfig.Clone()
				srvTLS.MinVersion = tls.VersionTLS13
				tlsConn := tls.QUICServer(&tls.QUICConfig{TLSConfig: srvTLS})
				tlsConn.SetTransportParameters(marshalTransportParams(serverCID))
				_ = tlsConn.Start(context.Background())

				conn = newQUICConn(l.udpConn, raddr, serverCID, clientSCID, true, tlsConn)
				wKeys, rKeys, _ := deriveInitialKeys(initialDCID, false)
				conn.writeKeys[0] = wKeys
				conn.readKeys[0] = rKeys
				l.conns[raddr.String()] = conn

				go func(c *quicConn) {
					select {
					case <-c.handshakeCh:
						l.acceptCh <- c
					case <-l.closed:
					}
				}(conn)
			}
		}
		l.mu.Unlock()

		if conn == nil {
			continue
		}
		pkts, err := DecodeCoalescedPackets(datagram, len(conn.localConnID), conn.getReadKeys)
		if err != nil {
			continue
		}
		for _, pkt := range pkts {
			conn.handlePacket(pkt)
		}
	}
}

func (l *quicListener) Accept(ctx context.Context) (Conn, error) {
	select {
	case c := <-l.acceptCh:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *quicListener) Addr() net.Addr { return l.udpConn.LocalAddr() }
func (l *quicListener) Close() error {
	close(l.closed)
	return l.udpConn.Close()
}

func DialQUIC(ctx context.Context, addr string, tlsConf *tls.Config) (Conn, error) {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	bindAddr := ":0"
	if raddr.IP.IsLoopback() {
		bindAddr = "127.0.0.1:0"
	}
	udpConn, err := net.ListenPacket("udp", bindAddr)
	if err != nil {
		return nil, err
	}

	host, _, _ := net.SplitHostPort(addr)
	conf := tlsConf.Clone()
	conf.MinVersion = tls.VersionTLS13
	if conf.ServerName == "" {
		conf.ServerName = host
	}
	if len(conf.NextProtos) == 0 {
		conf.NextProtos = []string{"h3"}
	}

	clientCID := randomConnID()
	initialDestCID := randomConnID()

	tlsConn := tls.QUICClient(&tls.QUICConfig{TLSConfig: conf})
	tlsConn.SetTransportParameters(marshalTransportParams(clientCID))

	conn := newQUICConn(udpConn, raddr, clientCID, initialDestCID, false, tlsConn)
	wKeys, rKeys, err := deriveInitialKeys(initialDestCID, true)
	if err != nil {
		udpConn.Close()
		return nil, err
	}
	conn.writeKeys[0] = wKeys
	conn.readKeys[0] = rKeys

	go func() {
		buf := make([]byte, 65535)
		for {
			n, _, err := udpConn.ReadFrom(buf)
			if err != nil {
				conn.cancel()
				return
			}
			pkts, err := DecodeCoalescedPackets(buf[:n], len(conn.localConnID), conn.getReadKeys)
			if err != nil {
				continue
			}
			for _, pkt := range pkts {
				conn.handlePacket(pkt)
			}
		}
	}()

	if err := tlsConn.Start(ctx); err != nil {
		udpConn.Close()
		return nil, err
	}
	if err := conn.drainTLSEvents(); err != nil {
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

func randomConnID() []byte {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return b
}

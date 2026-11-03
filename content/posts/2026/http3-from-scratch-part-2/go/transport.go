package main

import (
	"context"
	"crypto/tls"
	"io"
	"net"

	"github.com/quic-go/quic-go"
)

// HTTP/3 Unidirectional Stream Types (RFC 9114 Section 6.2 & RFC 9204 Section 4.2)
const (
	StreamTypeControl      uint64 = 0x00
	StreamTypePush         uint64 = 0x01
	StreamTypeQPACKEncoder uint64 = 0x02
	StreamTypeQPACKDecoder uint64 = 0x03
)

// HTTP/3 Settings Identifiers (RFC 9114 Section 7.2.4.1 & RFC 9204 Section 5)
const (
	SettingsQPACKMaxTableCapacity uint64 = 0x01
	SettingsMaxFieldSectionSize   uint64 = 0x06
	SettingsQPACKBlockedStreams   uint64 = 0x07
)

// Stream abstracts a bidirectional QUIC stream.
type Stream interface {
	io.ReadWriteCloser
	StreamID() int64
}

// ReceiveStream and SendStream abstract unidirectional QUIC streams.
type ReceiveStream interface {
	io.Reader
	StreamID() int64
}

type SendStream interface {
	io.WriteCloser
	StreamID() int64
}

// Conn abstracts a QUIC connection so our HTTP/3 server and client can run on
// quic-go today and our own from-scratch QUIC layer in Part 3!
type Conn interface {
	AcceptStream(ctx context.Context) (Stream, error)
	OpenStreamSync(ctx context.Context) (Stream, error)
	AcceptUniStream(ctx context.Context) (ReceiveStream, error)
	OpenUniStreamSync(ctx context.Context) (SendStream, error)
	RemoteAddr() net.Addr
	Context() context.Context
	CloseWithError(code uint64, desc string) error
}

// Listener accepts incoming QUIC connections.
type Listener interface {
	Accept(ctx context.Context) (Conn, error)
	Addr() net.Addr
	Close() error
}

type quicGoConn struct {
	*quic.Conn
}

func (c *quicGoConn) AcceptStream(ctx context.Context) (Stream, error) {
	s, err := c.Conn.AcceptStream(ctx)
	if err != nil {
		return nil, err
	}
	return &quicGoStream{s}, nil
}

func (c *quicGoConn) OpenStreamSync(ctx context.Context) (Stream, error) {
	s, err := c.Conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &quicGoStream{s}, nil
}

func (c *quicGoConn) AcceptUniStream(ctx context.Context) (ReceiveStream, error) {
	s, err := c.Conn.AcceptUniStream(ctx)
	if err != nil {
		return nil, err
	}
	return &quicGoRecvStream{s}, nil
}

func (c *quicGoConn) OpenUniStreamSync(ctx context.Context) (SendStream, error) {
	s, err := c.Conn.OpenUniStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &quicGoSendStream{s}, nil
}

func (c *quicGoConn) CloseWithError(code uint64, desc string) error {
	return c.Conn.CloseWithError(quic.ApplicationErrorCode(code), desc)
}

type quicGoStream struct{ *quic.Stream }

func (s *quicGoStream) StreamID() int64 { return int64(s.Stream.StreamID()) }

type quicGoRecvStream struct{ *quic.ReceiveStream }

func (s *quicGoRecvStream) StreamID() int64 { return int64(s.ReceiveStream.StreamID()) }

type quicGoSendStream struct{ *quic.SendStream }

func (s *quicGoSendStream) StreamID() int64 { return int64(s.SendStream.StreamID()) }

type quicGoListener struct {
	*quic.Listener
}

func (l *quicGoListener) Accept(ctx context.Context) (Conn, error) {
	c, err := l.Listener.Accept(ctx)
	if err != nil {
		return nil, err
	}
	return &quicGoConn{c}, nil
}

func ListenQUIC(addr string, tlsConf *tls.Config) (Listener, error) {
	l, err := quic.ListenAddr(addr, tlsConf, nil)
	if err != nil {
		return nil, err
	}
	return &quicGoListener{l}, nil
}

func DialQUIC(ctx context.Context, addr string, tlsConf *tls.Config) (Conn, error) {
	c, err := quic.DialAddr(ctx, addr, tlsConf, nil)
	if err != nil {
		return nil, err
	}
	return &quicGoConn{c}, nil
}

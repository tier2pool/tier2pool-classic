package jsonrpc

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"time"
)

type Conn interface {
	net.Conn

	Call(id int, method string, params interface{}) error
	SetReadDeadlineBySecond(second int) error
}

var _ Conn = &conn{}

type conn struct {
	netConn net.Conn
}

func (c *conn) Read(b []byte) (n int, err error) {
	return c.netConn.Read(b)
}

func (c *conn) Write(b []byte) (n int, err error) {
	return c.netConn.Write(append(b, []byte{'\n'}...))
}

func (c *conn) Close() error {
	return c.netConn.Close()
}

func (c *conn) LocalAddr() net.Addr {
	return c.netConn.LocalAddr()
}

func (c *conn) RemoteAddr() net.Addr {
	return c.netConn.RemoteAddr()
}

func (c *conn) SetDeadline(t time.Time) error {
	return c.netConn.SetDeadline(t)
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return c.netConn.SetReadDeadline(t)
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return c.netConn.SetWriteDeadline(t)
}

func (c *conn) Call(id int, method string, params interface{}) error {
	paramsData, err := json.Marshal(params)
	if err != nil {
		return err
	}

	request := Request{
		ID:     id,
		Method: method,
		Params: paramsData,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	_, err = c.netConn.Write(append(data, []byte{'\n'}...))

	return err
}

func (c *conn) SetReadDeadlineBySecond(second int) error {
	// Default 3 seconds timeout
	if second == 0 {
		second = 3
	}

	return c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(second)))
}

func Dial(rawURL string) (Conn, error) {
	remoteURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	var netConn net.Conn

	switch remoteURL.Scheme {
	case "tls", "ssl":
		netConn, err = tls.Dial("tcp", remoteURL.Host, &tls.Config{})
	case "tcp":
		netConn, err = net.Dial("tcp", remoteURL.Host)
	default:
		return nil, errors.New("scheme not support")
	}

	return &conn{
		netConn: netConn,
	}, nil
}

func New(netConn net.Conn) Conn {
	return &conn{
		netConn: netConn,
	}
}

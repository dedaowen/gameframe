package fserver

import (
	"io"
	"net"
	"time"

	"github.com/dedaowen/gameframe/fnet"
	"github.com/dedaowen/gameframe/iface"
	"github.com/dedaowen/gameframe/utils"
	"github.com/gorilla/websocket"
)

// WsConn websocket connection
type WsConn struct {
	conn   *websocket.Conn
	typ    int // message type
	reader io.Reader
}

// NewDefaultWsServer is default websocket server
func NewDefaultWsServer() iface.Iserver {
	s := &Server{
		Name:          utils.GlobalObject.Name,
		IPVersion:     "ws",
		IP:            "/ws",
		Port:          utils.GlobalObject.TcpPort,
		MaxConn:       utils.GlobalObject.MaxConn,
		connectionMgr: fnet.NewConnectionMgr(),
		Protoc:        utils.GlobalObject.Protoc,
		GenNum:        utils.NewUUIDGenerator(""),
	}
	utils.GlobalObject.TcpServer = s

	return s
}

// NewWsServer is websocket server
// @addr	格式：/addr，整体为 :port/addr
func NewWsServer(name string, addr string, port int, maxConn int, protoc iface.IServerProtocol) iface.Iserver {
	s := &Server{
		Name:          name,
		IPVersion:     "ws",
		IP:            addr,
		Port:          port,
		MaxConn:       maxConn,
		connectionMgr: fnet.NewConnectionMgr(),
		Protoc:        protoc,
		GenNum:        utils.NewUUIDGenerator(""),
	}
	utils.GlobalObject.TcpServer = s

	return s
}

// NewWSConn return an initialized *wsConn
func NewWSConn(conn *websocket.Conn) (*WsConn, error) {
	c := &WsConn{conn: conn}

	t, r, err := conn.NextReader()
	if err != nil {
		return nil, err
	}

	c.typ = t
	c.reader = r

	return c, nil
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *WsConn) Read(b []byte) (int, error) {
	n, err := c.reader.Read(b)
	if err != nil && err != io.EOF {
		return n, err
	} else if err == io.EOF {
		_, r, err := c.conn.NextReader()
		if err != nil {
			return 0, err
		}
		c.reader = r
	}

	return n, nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *WsConn) Write(b []byte) (int, error) {
	err := c.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *WsConn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address.
func (c *WsConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *WsConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *WsConn) SetDeadline(t time.Time) error {
	if err := c.conn.SetReadDeadline(t); err != nil {
		return err
	}

	return c.conn.SetWriteDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *WsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *WsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

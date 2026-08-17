package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

type Connection struct {
	conn   net.Conn
	read   *bufio.Reader
	enc    *json.Encoder
	writeM sync.Mutex
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{conn: conn, read: bufio.NewReader(conn), enc: json.NewEncoder(conn)}
}

func (c *Connection) ReadMessage() (Message, error) {
	var msg Message
	line, err := c.read.ReadBytes('\n')
	if err != nil { return msg, err }
	if len(line) > 64*1024 { return msg, fmt.Errorf("protocol message exceeds 64 KiB") }
	if err := json.Unmarshal(line, &msg); err != nil { return msg, fmt.Errorf("invalid protocol message: %w", err) }
	return msg, nil
}

func (c *Connection) WriteMessage(msg Message) error {
	c.writeM.Lock()
	defer c.writeM.Unlock()
	return c.enc.Encode(msg)
}

func (c *Connection) Close() error { return c.conn.Close() }

func ErrorResponse(id uint64, code, message string) Message {
	ok := false
	return Message{ID: id, Type: TypeResponse, OK: &ok, Error: &Error{Code: code, Message: message}}
}

func SuccessResponse(id uint64, service, action string, payload interface{}) Message {
	ok := true
	return Message{ID: id, Type: TypeResponse, Service: service, Action: action, OK: &ok, Payload: payload}
}

func IsClosed(err error) bool { return err == io.EOF || err == net.ErrClosed }

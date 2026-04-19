package aero

import (
	"errors"
	"sync"

	"github.com/Alsond5/aero/websocket"
)

var bufPool = sync.Pool{
	New: func() any {
		buf := make(buffer, 0, 4096)
		return &buf
	},
}

type buffer []byte

// WSConn is an Aero WebSocket connection. It wraps the underlying low-level
// connection and provides a high-level API for reading, writing, and
// connection-scoped storage. Obtain a WSConn inside a [WebSocketHandler].
type WSConn struct {
	conn           *websocket.Conn
	maxMessageSize uint64
	buf            *buffer
	locals         map[string]any
}

// Locals stores or retrieves a connection-scoped value by key. When called
// with a value argument, it sets the key and returns the new value.
// When called with the key only, it returns the current value or nil.
//
//	ws.Locals("userID", 42)
//	id := ws.Locals("userID").(int)
func (ws *WSConn) Locals(key string, value ...any) any {
	if len(value) > 0 {
		ws.locals[key] = value[0]
		return nil
	}

	return ws.locals[key]
}

// ReadMessage reads the next message from the connection.
// Returns the message type (aero.TextMessage or aero.BinaryMessage),
// the payload, and any error. A non-nil error typically means the
// connection was closed or the message exceeded [WSConfig.MaxMessageSize].
func (ws *WSConn) ReadMessage() (int, []byte, error) {
	ws.releaseBuf()

	var hdr websocket.Header
	if err := ws.conn.NextHeader(&hdr); err != nil {
		return 0, nil, err
	}

	firstOp := hdr.OpCode

	ws.acquireBuf()
	msgBuf := (*ws.buf)[:0]

	for {
		if uint64(len(msgBuf))+hdr.Length > ws.maxMessageSize {
			ws.conn.CloseWithError(websocket.CloseMessageTooBig, "message too big") //nolint:errcheck
			return 0, nil, errors.New("aero: maximum message size exceeded")
		}

		start := len(msgBuf)
		need := start + int(hdr.Length)

		if need > cap(msgBuf) {
			ws.releaseBuf()

			newBuf := make([]byte, need, max(need, cap(msgBuf)*2))
			copy(newBuf, msgBuf)
			msgBuf = newBuf
		}

		msgBuf = msgBuf[:need]

		if err := ws.conn.ReadPayload(hdr, msgBuf[start:]); err != nil {
			return 0, nil, err
		}

		if hdr.Fin {
			return int(firstOp), msgBuf, nil
		}

		if err := ws.conn.NextHeader(&hdr); err != nil {
			return 0, nil, err
		}
	}
}

// WriteMessage sends a message of the given type to the client.
// mt should be aero.TextMessage or aero.BinaryMessage.
func (ws *WSConn) WriteMessage(mt int, payload []byte) error {
	return ws.conn.WriteMessage(websocket.OpCode(mt), payload)
}

// Close sends a normal closure frame and closes the underlying connection.
func (ws *WSConn) Close() error {
	return ws.conn.Close()
}

// CloseWithReason sends a close frame with the given status code and reason
// string before closing the connection.
//
//	ws.CloseWithReason(websocket.CloseNormalClosure, "bye")
func (ws *WSConn) CloseWithReason(code websocket.CloseStatusCode, reason string) error {
	return ws.conn.CloseWithError(code, reason)
}

func (ws *WSConn) releaseBuf() {
	if ws.buf == nil {
		return
	}

	bufPool.Put(ws.buf)
	ws.buf = nil
}

func (ws *WSConn) acquireBuf() {
	if ws.buf != nil {
		return
	}

	ws.buf = bufPool.Get().(*buffer)
}

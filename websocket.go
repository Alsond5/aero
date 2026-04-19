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

type WSConn struct {
	conn           *websocket.Conn
	maxMessageSize uint64
	buf            *buffer
	locals         map[string]any
}

func (ws *WSConn) Locals(key string, value ...any) any {
	if len(value) > 0 {
		ws.locals[key] = value[0]
		return nil
	}

	return ws.locals[key]
}

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

func (ws *WSConn) WriteMessage(mt int, payload []byte) error {
	return ws.conn.WriteMessage(websocket.OpCode(mt), payload)
}

func (ws *WSConn) Close() error {
	return ws.conn.Close()
}

func (ws *WSConn) CloseWithReason(code websocket.CloseStaatusCode, reason string) error {
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

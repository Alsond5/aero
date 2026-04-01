package aero

import (
	"errors"
	"sync"

	"github.com/Alsond5/aero/websocket"
)

const (
	DefaultBufSize = 4 << 10
	MaxIdleBufSize = 64 << 10
)

var wsPool = sync.Pool{
	New: func() any {
		return &Websocket{
			msgBuf: make([]byte, 0, DefaultBufSize),
			locals: make(map[string]any),
		}
	},
}

type Websocket struct {
	conn           *websocket.Conn
	msgBuf         []byte
	maxMessageSize uint64
	locals         map[string]any
}

func (ws *Websocket) reset() {
	if cap(ws.msgBuf) > MaxIdleBufSize {
		ws.msgBuf = make([]byte, 0, DefaultBufSize)
	} else {
		ws.msgBuf = ws.msgBuf[:0]
	}
	ws.conn = nil
	ws.maxMessageSize = 0
	clear(ws.locals)
}

func (ws *Websocket) Locals(key string, value ...any) any {
	if len(value) > 0 {
		ws.locals[key] = value[0]
		return nil
	}

	return ws.locals[key]
}

func (ws *Websocket) ReadMessage() (websocket.OpCode, []byte, error) {
	ws.msgBuf = ws.msgBuf[:0]

	var firstOp websocket.OpCode
	var isFirst = true

	var hdr websocket.Header

	for {
		err := ws.conn.NextHeader(&hdr)
		if err != nil {
			return 0, nil, err
		}

		if uint64(len(ws.msgBuf))+hdr.Length > ws.maxMessageSize {
			ws.conn.CloseWithError(websocket.CloseMessageTooBig, "message too big")
			return 0, nil, errors.New("aero: maximum message size exceeded")
		}

		if isFirst {
			firstOp = hdr.OpCode
			isFirst = false
		}

		start := len(ws.msgBuf)
		need := start + int(hdr.Length)

		if need > cap(ws.msgBuf) {
			newBuf := make([]byte, need, max(need, cap(ws.msgBuf)*2))
			copy(newBuf, ws.msgBuf)
			ws.msgBuf = newBuf
		} else {
			ws.msgBuf = ws.msgBuf[:need]
		}

		if err := ws.conn.ReadPayload(hdr, ws.msgBuf[start:]); err != nil {
			return 0, nil, err
		}

		if hdr.Fin {
			return firstOp, ws.msgBuf, nil
		}
	}
}

func (ws *Websocket) WriteMessage(op websocket.OpCode, payload []byte) error {
	return ws.conn.WriteMessage(op, payload)
}

func (ws *Websocket) Close() error {
	return ws.conn.Close()
}

func (ws *Websocket) CloseWithReason(code websocket.CloseStaatusCode, reason string) error {
	return ws.conn.CloseWithError(code, reason)
}

package aero

import (
	"errors"

	"github.com/Alsond5/aero/websocket"
)

type WSMessageType string

const (
	WSMessageText   WSMessageType = "text"
	WSMessageBinary WSMessageType = "binary"
)

func opToMsgTyp(op websocket.OpCode) WSMessageType {
	if !op.IsData() {
		return ""
	}

	switch op {
	case websocket.OpText:
		return WSMessageText
	case websocket.OpBinary:
		return WSMessageBinary
	}

	return ""
}

func msgTypToOp(mt WSMessageType) websocket.OpCode {
	switch mt {
	case WSMessageText:
		return websocket.OpText
	case WSMessageBinary:
		return websocket.OpBinary
	}

	return websocket.OpText
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

func (ws *WSConn) ReadMessage() (WSMessageType, []byte, error) {
	*ws.buf = (*ws.buf)[:0]
	msgBuf := *ws.buf

	var firstOp websocket.OpCode
	var isFirst = true
	var hdr websocket.Header

	for {
		err := ws.conn.NextHeader(&hdr)
		if err != nil {
			return "", nil, err
		}

		if uint64(len(msgBuf))+hdr.Length > ws.maxMessageSize {
			ws.conn.CloseWithError(websocket.CloseMessageTooBig, "message too big")
			return "", nil, errors.New("aero: maximum message size exceeded")
		}

		if isFirst {
			firstOp = hdr.OpCode
			isFirst = false
		}

		start := len(msgBuf)
		need := start + int(hdr.Length)

		if need > cap(msgBuf) {
			newBuf := make([]byte, need, max(need, cap(msgBuf)*2))
			copy(newBuf, msgBuf)
			msgBuf = newBuf
		}

		msgBuf = msgBuf[:need]

		if err := ws.conn.ReadPayload(hdr, msgBuf[start:]); err != nil {
			return "", nil, err
		}

		if hdr.Fin {
			return opToMsgTyp(firstOp), msgBuf, nil
		}
	}
}

func (ws *WSConn) WriteMessage(mt WSMessageType, payload []byte) error {
	return ws.conn.WriteMessage(msgTypToOp(mt), payload)
}

func (ws *WSConn) Close() error {
	return ws.conn.Close()
}

func (ws *WSConn) CloseWithReason(code websocket.CloseStaatusCode, reason string) error {
	return ws.conn.CloseWithError(code, reason)
}

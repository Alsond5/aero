// websocket/frame.go

package websocket

import (
	"encoding/binary"
)

type OpCode byte

const (
	OpContinuation OpCode = 0x0
	OpText         OpCode = 0x1
	OpBinary       OpCode = 0x2
	// 0x3–0x7: reserved data frames
	OpClose OpCode = 0x8
	OpPing  OpCode = 0x9
	OpPong  OpCode = 0xA
	// 0xB–0xF: reserved control frames
)

func (op OpCode) IsControl() bool { return op&0x8 != 0 }
func (op OpCode) IsData() bool    { return op&0x8 == 0 }
func (op OpCode) IsReserved() bool {
	return (0x3 <= op && op <= 0x7) || (0xb <= op && op <= 0xf)
}

const maxControlPayload = 125

type Header struct {
	Fin    bool
	Rsv    byte
	OpCode OpCode
	Masked bool
	Mask   [4]byte
	Length uint64
}

func (h Header) RSV1() bool { return h.Rsv&0x4 != 0 }
func (h Header) RSV2() bool { return h.Rsv&0x2 != 0 }
func (h Header) RSV3() bool { return h.Rsv&0x1 != 0 }

func (h Header) IsControl() bool { return h.OpCode.IsControl() }

func RsvBits(rsv byte) (r1, r2, r3 bool) {
	r1 = rsv&0x04 != 0
	r2 = rsv&0x02 != 0
	r3 = rsv&0x01 != 0

	return r1, r2, r3
}

type Frame struct {
	Header  Header
	Payload []byte
}

func NewFrame(op OpCode, fin bool, p []byte) Frame {
	return Frame{
		Header: Header{
			Fin:    fin,
			OpCode: op,
			Length: uint64(len(p)),
		},
		Payload: p,
	}
}

func NewTextFrame(p []byte) Frame {
	return NewFrame(OpText, true, p)
}

func NewBinaryFrame(p []byte) Frame {
	return NewFrame(OpBinary, true, p)
}

func NewPingFrame(p []byte) Frame {
	return NewFrame(OpPing, true, p)
}

func NewPongFrame(p []byte) Frame {
	return NewFrame(OpPong, true, p)
}

func NewCloseFrame(p []byte) Frame {
	return NewFrame(OpClose, true, p)
}

func NewCloseFrameWithReason(code CloseStaatusCode, reason string) Frame {
	n := min(2+len(reason), maxControlPayload)
	p := make([]byte, n)

	crop := min(maxControlPayload-2, len(reason))
	PutCloseFrameBody(p, code, reason[:crop])

	return NewFrame(OpClose, true, p)
}

func PutCloseFrameBody(p []byte, code CloseStaatusCode, reason string) {
	_ = p[1+len(reason)]
	binary.BigEndian.PutUint16(p, code.Value())
	copy(p[2:], reason)
}

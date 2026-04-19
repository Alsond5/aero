// websocket/frame.go

package websocket

import (
	"encoding/binary"
)

// OpCode represents a WebSocket frame opcode as defined in RFC 6455 §5.2.
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

// IsControl reports whether op is a control opcode (Close, Ping, or Pong).
func (op OpCode) IsControl() bool { return op&0x8 != 0 }

// IsData reports whether op is a data opcode (Text or Binary).
func (op OpCode) IsData() bool { return op&0x8 == 0 }

// IsReserved reports whether op is a reserved opcode not defined by RFC 6455.
func (op OpCode) IsReserved() bool {
	return (0x3 <= op && op <= 0x7) || (0xb <= op && op <= 0xf)
}

const maxControlPayload = 125

// Header represents a parsed WebSocket frame header.
type Header struct {
	Fin    bool    // FIN bit - true if this is the final fragment.
	Rsv    byte    // RSV1-RSV3 bits packed into the low 3 bits.
	OpCode OpCode  // Frame opcode.
	Masked bool    // Whether the payload is masked.
	Mask   [4]byte // Masking key (only meaningful when Masked is true).
	Length uint64  // Payload length in bytes.
}

// RSV1 reports whether the RSV1 extension bit is set in the frame header.
func (h Header) RSV1() bool { return h.Rsv&0x4 != 0 }

// RSV2 reports whether the RSV2 extension bit is set in the frame header.
func (h Header) RSV2() bool { return h.Rsv&0x2 != 0 }

// RSV3 reports whether the RSV3 extension bit is set in the frame header.
func (h Header) RSV3() bool { return h.Rsv&0x1 != 0 }

// IsControl reports whether the frame header describes a control frame.
func (h Header) IsControl() bool { return h.OpCode.IsControl() }

// RsvBits unpacks the three RSV bits from a raw rsv byte into individual booleans.
func RsvBits(rsv byte) (r1, r2, r3 bool) {
	r1 = rsv&0x04 != 0
	r2 = rsv&0x02 != 0
	r3 = rsv&0x01 != 0

	return r1, r2, r3
}

// Frame is a complete WebSocket frame ready to be written to the wire.
type Frame struct {
	Header  Header
	Payload []byte
}

// NewFrame constructs a WebSocket frame with the given opcode, FIN bit, and payload.
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

// NewTextFrame constructs a FIN text frame with the given UTF-8 payload.
func NewTextFrame(p []byte) Frame {
	return NewFrame(OpText, true, p)
}

// NewBinaryFrame constructs a FIN binary frame with the given payload.
func NewBinaryFrame(p []byte) Frame {
	return NewFrame(OpBinary, true, p)
}

// NewPingFrame constructs a ping control frame with an optional payload (max 125 bytes).
func NewPingFrame(p []byte) Frame {
	return NewFrame(OpPing, true, p)
}

// NewPongFrame constructs a pong control frame with an optional payload (max 125 bytes).
func NewPongFrame(p []byte) Frame {
	return NewFrame(OpPong, true, p)
}

// NewCloseFrame constructs a close frame with a raw pre-encoded payload.
// Prefer [NewCloseFrameWithReason] when constructing close frames manually.
func NewCloseFrame(p []byte) Frame {
	return NewFrame(OpClose, true, p)
}

// NewCloseFrameWithReason constructs a close frame with the given status
// code and reason string encoded into the payload per RFC 6455 §5.5.1.
func NewCloseFrameWithReason(code CloseStatusCode, reason string) Frame {
	n := min(2+len(reason), maxControlPayload)
	p := make([]byte, n)

	crop := min(maxControlPayload-2, len(reason))
	PutCloseFrameBody(p, code, reason[:crop])

	return NewFrame(OpClose, true, p)
}

// PutCloseFrameBody encodes the given status code and reason string into p
// following the RFC 6455 close frame body layout. p must be at least
// 2 + len(reason) bytes.
func PutCloseFrameBody(p []byte, code CloseStatusCode, reason string) {
	_ = p[1+len(reason)]
	binary.BigEndian.PutUint16(p, code.Value())
	copy(p[2:], reason)
}

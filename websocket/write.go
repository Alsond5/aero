package websocket

import (
	"bufio"
)

const (
	MaxHeaderSize  = 14
	MinHeaderSize  = 2
	MaxFrameLength = 1<<63 - 1
)

// WriteHeader encodes and writes a WebSocket frame header to w.
func WriteHeader(w *bufio.Writer, h Header) error {
	b0 := h.Rsv<<4 | byte(h.OpCode)
	if h.Fin {
		b0 |= 0x80
	}
	if err := w.WriteByte(b0); err != nil {
		return err
	}

	var b1 byte
	switch {
	case h.Length > MaxFrameLength:
		return ErrHeaderLengthUnexpected
	case h.Length > 65535:
		b1 = 127
	case h.Length > 125:
		b1 = 126
	default:
		b1 = byte(h.Length)
	}

	if h.Masked {
		b1 |= 0x80
	}
	if err := w.WriteByte(b1); err != nil {
		return err
	}

	switch {
	case h.Length > 65535:
		l := h.Length
		for i := 7; i >= 0; i-- {
			if err := w.WriteByte(byte(l >> (uint(i) * 8))); err != nil {
				return err
			}
		}
	case h.Length > 125:
		if err := w.WriteByte(byte(h.Length >> 8)); err != nil {
			return err
		}
		if err := w.WriteByte(byte(h.Length)); err != nil {
			return err
		}
	}

	if h.Masked {
		for i := range 4 {
			if err := w.WriteByte(h.Mask[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// WriteFrame encodes and writes a complete WebSocket frame to w.
func WriteFrame(w *bufio.Writer, f Frame) error {
	err := WriteHeader(w, f.Header)
	if err != nil {
		return err
	}

	_, err = w.Write(f.Payload)
	return err
}

package websocket

import (
	"encoding/binary"
	"io"
)

const (
	MaxHeaderSize  = 14
	MinHeaderSize  = 2
	MaxFrameLength = 1<<63 - 1
)

func WriteHeader(w io.Writer, h Header) error {
	var buf [MaxHeaderSize]byte
	buf[0] = h.Rsv<<4 | byte(h.OpCode)
	if h.Fin {
		buf[0] |= 0x80
	}

	var n int
	switch {
	case h.Length > MaxFrameLength:
		return ErrHeaderLengthUnexpected
	case h.Length > 65535:
		buf[1] = 127
		binary.BigEndian.PutUint64(buf[2:10], h.Length)
		n = 10
	case h.Length > 125:
		buf[1] = 126
		binary.BigEndian.PutUint16(buf[2:4], uint16(h.Length))
		n = 4
	default:
		buf[1] = byte(h.Length)
		n = 2
	}

	if h.Masked {
		buf[1] |= 0x80
		n += copy(buf[n:], h.Mask[:])
	}

	_, err := w.Write(buf[:n])

	return err
}

func WriteFrame(w io.Writer, f Frame) error {
	err := WriteHeader(w, f.Header)
	if err != nil {
		return err
	}

	_, err = w.Write(f.Payload)
	return err
}

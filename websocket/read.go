package websocket

import (
	"bufio"
	"encoding/binary"
)

// ReadHeader reads and parses a WebSocket frame header from r into dst.
// Returns an error if the stream is malformed or the connection is closed.
func ReadHeader(r *bufio.Reader, dst *Header) error {
	b0, err := r.ReadByte()
	if err != nil {
		return err
	}

	b1, err := r.ReadByte()
	if err != nil {
		return err
	}

	dst.Fin = b0&0x80 != 0
	dst.Rsv = (b0 >> 4) & 0x7
	dst.OpCode = OpCode(b0 & 0x0F)
	dst.Masked = b1&0x80 != 0

	length := uint64(b1 & 0x7F)
	switch {
	case length < 126:
		dst.Length = length

	case length == 126:
		b0, err = r.ReadByte()
		if err != nil {
			return err
		}
		b1, err = r.ReadByte()
		if err != nil {
			return err
		}
		dst.Length = uint64(b0)<<8 | uint64(b1)

	case length == 127:
		var l uint64
		for range 8 {
			b, err := r.ReadByte()
			if err != nil {
				return err
			}
			l = l<<8 | uint64(b)
		}
		if l&(1<<63) != 0 {
			return ErrHeaderLengthUnexpected
		}
		dst.Length = l

	default:
		return ErrHeaderLengthUnexpected
	}

	if dst.Masked {
		for i := range 4 {
			b, err := r.ReadByte()
			if err != nil {
				return err
			}
			dst.Mask[i] = b
		}
	}

	return nil
}

// ParseCloseFrameData decodes the status code and reason string from a
// close frame payload. Returns CloseNormalClosure with an empty reason
// if the payload is empty.
func ParseCloseFrameData(payload []byte) (code CloseStatusCode, reason string) {
	if len(payload) < 2 {
		return code, reason
	}
	code = CloseStatusCode(binary.BigEndian.Uint16(payload))
	reason = string(payload[2:])

	return code, reason
}

// ParseCloseFrameDataUnsafe is identical to [ParseCloseFrameData] but avoids
// a heap allocation by returning the reason as a string backed by the
// original payload slice. The caller must not retain the payload after
// the returned string is used.
func ParseCloseFrameDataUnsafe(payload []byte) (code CloseStatusCode, reason string) {
	if len(payload) < 2 {
		return code, reason
	}
	code = CloseStatusCode(binary.BigEndian.Uint16(payload))
	reason = unsafeByteToString(payload[2:])

	return code, reason
}

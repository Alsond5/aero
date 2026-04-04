package websocket

import (
	"bufio"
	"encoding/binary"
)

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

func ParseCloseFrameData(payload []byte) (code CloseStaatusCode, reason string) {
	if len(payload) < 2 {
		return code, reason
	}
	code = CloseStaatusCode(binary.BigEndian.Uint16(payload))
	reason = string(payload[2:])

	return code, reason
}

func ParseCloseFrameDataUnsafe(payload []byte) (code CloseStaatusCode, reason string) {
	if len(payload) < 2 {
		return code, reason
	}
	code = CloseStaatusCode(binary.BigEndian.Uint16(payload))
	reason = unsafeByteToString(payload[2:])

	return code, reason
}

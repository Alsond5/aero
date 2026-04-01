package websocket

import (
	"encoding/binary"
	"io"
	"unsafe"
)

func ReadHeader(r io.Reader, dst *Header) error {
	var buf [MaxHeaderSize]byte
	if _, err := io.ReadFull(r, buf[:2]); err != nil {
		return err
	}

	dst.Fin = buf[0]&0x80 != 0
	dst.Rsv = (buf[0] >> 4) & 0x7
	dst.OpCode = OpCode(buf[0] & 0x0F)
	dst.Masked = buf[1]&0x80 != 0

	length := uint64(buf[1] & 0x7F)
	switch {
	case length < uint64(126):
		dst.Length = length
	case length == 126:
		if _, err := io.ReadFull(r, buf[:2]); err != nil {
			return err
		}

		dst.Length = uint64(binary.BigEndian.Uint16(buf[:2]))
	case length == 127:
		if _, err := io.ReadFull(r, buf[:8]); err != nil {
			return err
		}
		if buf[0]&0x80 != 0 {
			return ErrHeaderLengthUnexpected
		}

		dst.Length = binary.BigEndian.Uint64(buf[:8])
	default:
		return ErrHeaderLengthUnexpected
	}

	if dst.Masked {
		if _, err := io.ReadFull(r, dst.Mask[:]); err != nil {
			return err
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

func unsafeByteToString(buf []byte) (str string) {
	return *(*string)(unsafe.Pointer(&buf))
}

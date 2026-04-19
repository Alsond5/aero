package websocket

import (
	"encoding/binary"
	"unsafe"
)

// Mask applies the WebSocket masking algorithm in-place on b using the
// 4-byte key. It processes 8 bytes at a time via unsafe uint64 XOR for
// maximum throughput, falling back to byte-by-byte for the remainder.
// Masking and unmasking use the same operation, so calling Mask twice
// restores the original data.
func Mask(key [4]byte, b []byte) {
	if len(b) == 0 {
		return
	}

	k32 := binary.LittleEndian.Uint32(key[:])
	k64 := uint64(k32)<<32 | uint64(k32)

	i := 0
	for ; i+8 <= len(b); i += 8 {
		v := *(*uint64)(unsafe.Pointer(&b[i]))
		*(*uint64)(unsafe.Pointer(&b[i])) = v ^ k64
	}

	for j := i; j < len(b); j++ {
		b[j] ^= key[j%4]
	}
}

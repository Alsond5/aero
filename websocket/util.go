package websocket

import "unsafe"

func unsafeByteToString(buf []byte) (str string) {
	return *(*string)(unsafe.Pointer(&buf))
}

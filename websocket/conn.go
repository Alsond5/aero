package websocket

import (
	"bufio"
	"io"
	"net"
	"sync"
)

type Conn struct {
	nc      net.Conn
	br      *bufio.Reader
	bw      *bufio.Writer
	writeMu sync.Mutex
}

func NewConn(nc net.Conn, br *bufio.Reader, bw *bufio.Writer) *Conn {
	return &Conn{
		nc: nc,
		br: br,
		bw: bw,
	}
}

func (c *Conn) NextHeader(hdr *Header) error {
	for {
		if err := ReadHeader(c.br, hdr); err != nil {
			return err
		}

		if hdr.IsControl() {
			if err := c.handleControl(hdr); err != nil {
				return err
			}
			continue
		}

		return nil
	}
}

func (c *Conn) ReadPayload(hdr Header, dst []byte) error {
	if hdr.Length == 0 {
		return nil
	}

	if _, err := io.ReadFull(c.br, dst); err != nil {
		return err
	}

	if hdr.Masked {
		Mask(hdr.Mask, dst)
	}

	return nil
}

func (c *Conn) WriteMessage(op OpCode, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return c.writeFrameUnsafe(op, payload)
}

func (c *Conn) Close() error {
	var buf [2]byte
	buf[0] = 0x03
	buf[1] = 0xE8

	c.writeMu.Lock()
	c.writeFrameUnsafe(OpClose, buf[:])
	c.writeMu.Unlock()

	return c.nc.Close()
}

func (c *Conn) CloseWithError(code CloseStaatusCode, reason string) error {
	var buf [maxControlPayload]byte
	buf[0] = byte(code >> 8)
	buf[1] = byte(code)

	n := copy(buf[2:], reason)

	c.writeMu.Lock()
	c.writeFrameUnsafe(OpClose, buf[:n+2])
	c.writeMu.Unlock()

	return c.nc.Close()
}

func (c *Conn) handleControl(hdr *Header) error {
	var buf [maxControlPayload]byte
	payload := buf[:hdr.Length]

	if hdr.Length > 0 {
		if _, err := io.ReadFull(c.br, payload); err != nil {
			return err
		}
		if hdr.Masked {
			Mask(hdr.Mask, payload)
		}
	}

	switch hdr.OpCode {
	case OpPing:
		c.writeMu.Lock()
		err := c.writeFrameUnsafe(OpPong, payload)
		c.writeMu.Unlock()

		return err

	case OpClose:
		c.writeMu.Lock()
		c.writeFrameUnsafe(OpClose, payload)
		c.writeMu.Unlock()

		if hdr.Length >= 2 {
			code, reason := ParseCloseFrameDataUnsafe(payload)
			return NewCloseError(code, reason)
		}

		return NewCloseError(CloseNormalClosure, "")

	case OpPong:
		return nil
	}

	return nil
}

func (c *Conn) writeFrameUnsafe(op OpCode, payload []byte) error {
	if err := WriteFrame(c.bw, NewFrame(op, true, payload)); err != nil {
		return err
	}

	return c.bw.Flush()
}

package websocket

import (
	"bufio"
	"io"
	"net"
	"sync"
)

// Conn is a WebSocket connection over a raw [net.Conn]. It provides
// frame-level read/write primitives. For a higher-level API, use
// [aero.WSConn] via [aero.WebSocket].
type Conn struct {
	nc      net.Conn
	br      *bufio.Reader
	bw      *bufio.Writer
	writeMu sync.Mutex
}

// NewConn wraps an established net.Conn with buffered reader and writer
// into a WebSocket Conn ready for frame exchange.
func NewConn(nc net.Conn, br *bufio.Reader, bw *bufio.Writer) *Conn {
	return &Conn{
		nc: nc,
		br: br,
		bw: bw,
	}
}

// NextHeader reads and returns the next frame header from the connection
// into hdr. Must be called before [Conn.ReadPayload] for each frame.
// Returns an error if the connection is closed or the frame is malformed.
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

// ReadPayload reads the payload of the frame described by hdr into dst.
// dst must be pre-allocated to at least hdr.Length bytes.
// Unmasking is applied automatically if the frame is masked.
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

// WriteMessage sends a single unfragmented message with the given opcode
// and payload. It is safe to call from a single goroutine at a time.
func (c *Conn) WriteMessage(op OpCode, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return c.writeFrameUnsafe(op, payload)
}

// Close sends a normal closure frame and closes the underlying connection.
func (c *Conn) Close() error {
	var buf [2]byte
	buf[0] = 0x03
	buf[1] = 0xE8

	c.writeMu.Lock()
	c.writeFrameUnsafe(OpClose, buf[:]) //nolint:errcheck
	c.writeMu.Unlock()

	return c.nc.Close()
}

// CloseWithError sends a close frame with the given status code and reason
// before closing the underlying connection.
func (c *Conn) CloseWithError(code CloseStatusCode, reason string) error {
	var buf [maxControlPayload]byte
	buf[0] = byte(code >> 8)
	buf[1] = byte(code)

	n := copy(buf[2:], reason)

	c.writeMu.Lock()
	c.writeFrameUnsafe(OpClose, buf[:n+2]) //nolint:errcheck
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
		c.writeFrameUnsafe(OpClose, payload) //nolint:errcheck
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

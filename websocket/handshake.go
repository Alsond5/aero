package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"strings"
	"time"
)

const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type Upgrader struct {
	CheckOrigin  func(origin string) bool
	Subprotocols []string
	WriteTimeout time.Duration
}

type Handshake struct {
	protocol string
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, ErrHijackNotSupport
	}

	nc, brw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	nc.SetDeadline(time.Time{})

	hs, err := u.validate(r)
	if err != nil {
		writeHTTPError(brw.Writer, err)
		nc.Close()

		return nil, err
	}

	if t := u.WriteTimeout; t != 0 {
		nc.SetWriteDeadline(time.Now().Add(t))
		defer nc.SetWriteDeadline(time.Time{})
	}

	clientKey := r.Header.Get("Sec-WebSocket-Key")
	if err := write101(brw.Writer, clientKey, hs.protocol); err != nil {
		nc.Close()
		return nil, err
	}

	return NewConn(nc, brw.Reader, brw.Writer), nil
}

func (u *Upgrader) validate(r *http.Request) (Handshake, error) {
	var hs Handshake

	if r.Method != http.MethodGet {
		return hs, ErrBadMethod
	}

	if r.ProtoMajor < 1 || (r.ProtoMajor == 1 && r.ProtoMinor < 1) {
		return hs, ErrBadProtocol
	}

	if r.Host == "" {
		return hs, ErrBadHost
	}

	if !headerContainsToken(r.Header, "Upgrade", "websocket") {
		return hs, ErrBadUpgrade
	}

	if !headerContainsToken(r.Header, "Connection", "upgrade") {
		return hs, ErrBadConnection
	}

	if len(r.Header.Get("Sec-WebSocket-Key")) != 24 {
		return hs, ErrBadSecKey
	}

	if v := r.Header.Get("Sec-WebSocket-Version"); v != "13" {
		if v != "" {
			return hs, ErrUpgradeRequired
		}
		return hs, ErrBadSecVersion
	}

	if u.CheckOrigin != nil && !u.CheckOrigin(r.Header.Get("Origin")) {
		return hs, ErrForbiddenOrigin
	}

	hs.protocol = u.negotiateSubprotocol(r)

	return hs, nil
}

func (u *Upgrader) negotiateSubprotocol(r *http.Request) string {
	if len(u.Subprotocols) == 0 {
		return ""
	}
	clientProtos := r.Header.Get("Sec-WebSocket-Protocol")
	if clientProtos == "" {
		return ""
	}

	for _, serverProto := range u.Subprotocols {
		if tokenListContains(clientProtos, serverProto) {
			return serverProto
		}
	}

	return ""
}

func write101(bw *bufio.Writer, clientKey, protocol string) error {
	var keyBuf [60]byte
	copy(keyBuf[:24], clientKey)
	copy(keyBuf[24:], magicGUID)

	digest := sha1.Sum(keyBuf[:])

	var acceptBuf [28]byte
	base64.StdEncoding.Encode(acceptBuf[:], digest[:])

	bw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	bw.WriteString("Upgrade: websocket\r\n")
	bw.WriteString("Connection: Upgrade\r\n")
	bw.WriteString("Sec-WebSocket-Accept: ")
	bw.Write(acceptBuf[:])
	bw.WriteString("\r\n")

	if protocol != "" {
		bw.WriteString("Sec-WebSocket-Protocol: ")
		bw.WriteString(protocol)
		bw.WriteString("\r\n")
	}

	bw.WriteString("\r\n")

	return bw.Flush()
}

func writeHTTPError(bw *bufio.Writer, err error) {
	code := errorStatusCode(err)
	status := http.StatusText(code)

	bw.WriteString("HTTP/1.1 ")
	writeDecimal(bw, code)
	bw.WriteByte(' ')
	bw.WriteString(status)
	bw.WriteString("\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	_ = bw.Flush()
}

func writeDecimal(bw *bufio.Writer, n int) {
	bw.WriteByte(byte('0' + (n/100)%10))
	bw.WriteByte(byte('0' + (n/10)%10))
	bw.WriteByte(byte('0' + n%10))
}

func headerContainsToken(h http.Header, key, token string) bool {
	for _, v := range h[http.CanonicalHeaderKey(key)] {
		if tokenListContains(v, token) {
			return true
		}
	}

	return false
}

func tokenListContains(list, token string) bool {
	for len(list) > 0 {
		i := strings.IndexByte(list, ',')
		var part string
		if i < 0 {
			part = list
			list = ""
		} else {
			part = list[:i]
			list = list[i+1:]
		}

		part = strings.TrimSpace(part)
		if strings.EqualFold(part, token) {
			return true
		}
	}

	return false
}

func errorStatusCode(err error) int {
	switch err {
	case ErrUpgradeRequired:
		return http.StatusUpgradeRequired
	case ErrForbiddenOrigin:
		return http.StatusForbidden
	case ErrBadMethod:
		return http.StatusMethodNotAllowed
	default:
		return http.StatusBadRequest
	}
}

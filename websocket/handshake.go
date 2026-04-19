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

var handshakeHeader = []byte(
	"HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: ",
)

var httpErrorHeader = []byte("HTTP/1.1 ")
var httpErrorFooter = []byte("\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")

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

	if err := nc.SetDeadline(time.Time{}); err != nil {
		return nil, err
	}

	hs, err := u.validate(r)
	if err != nil {
		writeHTTPError(brw.Writer, err)
		nc.Close() //nolint:errcheck

		return nil, err
	}

	if t := u.WriteTimeout; t != 0 {
		if err := nc.SetWriteDeadline(time.Now().Add(t)); err != nil {
			nc.Close() //nolint:errcheck
			return nil, err
		}

		defer nc.SetWriteDeadline(time.Time{}) //nolint:errcheck
	}

	clientKey := r.Header.Get("Sec-WebSocket-Key")
	if err := write101(brw.Writer, clientKey, hs.protocol); err != nil {
		nc.Close() //nolint:errcheck
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

	bw.Write(handshakeHeader) //nolint:errcheck
	bw.Write(acceptBuf[:])    //nolint:errcheck
	bw.WriteString("\r\n")    //nolint:errcheck

	if protocol != "" {
		bw.WriteString("Sec-WebSocket-Protocol: ") //nolint:errcheck
		bw.WriteString(protocol)                   //nolint:errcheck
		bw.WriteString("\r\n")                     //nolint:errcheck
	}

	bw.WriteString("\r\n") //nolint:errcheck

	return bw.Flush()
}

func writeHTTPError(bw *bufio.Writer, err error) {
	code := errorStatusCode(err)
	status := http.StatusText(code)

	bw.Write(httpErrorHeader) //nolint:errcheck
	writeDecimal(bw, code)
	bw.WriteByte(' ')         //nolint:errcheck
	bw.WriteString(status)    //nolint:errcheck
	bw.Write(httpErrorFooter) //nolint:errcheck

	_ = bw.Flush()
}

func writeDecimal(bw *bufio.Writer, n int) {
	_ = bw.WriteByte(byte('0' + (n/100)%10))
	_ = bw.WriteByte(byte('0' + (n/10)%10))
	_ = bw.WriteByte(byte('0' + n%10))
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

package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
)

const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

var upgradeResponse = "HTTP/1.1 101 Switching Protocols\r\n" +
	"Upgrade: websocket\r\n" +
	"Connection: Upgrade\r\n" +
	"Sec-WebSocket-Accept: "

type Upgrader struct {
	CheckOrigin  func(origin string) bool
	Subprotocols []string
	Extensions   []string
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if !isWebSocketUpgrade(r) {
		return nil, ErrNotWebSocket
	}

	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return nil, ErrBadWebSocketVersion
	}

	clientKey := r.Header.Get("Sec-WebSocket-Key")
	if err := validateKey(clientKey); err != nil {
		return nil, err
	}

	if u.CheckOrigin != nil {
		origin := r.Header.Get("Origin")
		if !u.CheckOrigin(origin) {
			return nil, ErrForbiddenOrigin
		}
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("websocket: response does not support hijacking")
	}

	nc, brw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	protocol := u.negotiateSubprotocol(r)
	extensions := u.negotiateExtensions(r)

	if err := write101(brw.Writer, clientKey, protocol, extensions); err != nil {
		nc.Close()
		return nil, err
	}

	return NewConn(nc, brw.Reader, brw.Writer), nil
}

func isWebSocketUpgrade(r *http.Request) bool {
	return r.Method == http.MethodGet &&
		headerContains(r.Header, "Connection", "upgrade") &&
		headerContains(r.Header, "Upgrade", "websocket")
}

func validateKey(key string) error {
	if key == "" {
		return ErrBadWebSocketKey
	}
	if len(key) != 24 {
		return ErrBadWebSocketKeyLen
	}

	var decoded [16]byte
	n, err := base64.StdEncoding.Decode(decoded[:], []byte(key))
	if err != nil || n != 16 {
		return ErrBadWebSocketKeyLen
	}

	return nil
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
		for _, clientProto := range splitHeader(clientProtos) {
			if serverProto == clientProto {
				return serverProto
			}
		}
	}

	return ""
}

func (u *Upgrader) negotiateExtensions(r *http.Request) string {
	if len(u.Extensions) == 0 {
		return ""
	}
	clientExts := r.Header.Get("Sec-WebSocket-Extensions")
	if clientExts == "" {
		return ""
	}
	for _, serverExt := range u.Extensions {
		for _, clientExt := range splitHeader(clientExts) {
			if serverExt == clientExt {
				return serverExt
			}
		}
	}
	return ""
}

func write101(bw *bufio.Writer, clientKey, protocol, extensions string) error {
	bw.WriteString(upgradeResponse)

	enc := base64.NewEncoder(base64.StdEncoding, bw)

	h := sha1.New()
	io.WriteString(h, clientKey)
	io.WriteString(h, magicGUID)

	var digest [sha1.Size]byte
	h.Sum(digest[:0])
	enc.Write(digest[:])
	enc.Close()

	bw.WriteString("\r\n")

	if protocol != "" {
		bw.WriteString("Sec-WebSocket-Protocol: ")
		bw.WriteString(protocol)
		bw.WriteString("\r\n")
	}

	if extensions != "" {
		bw.WriteString("Sec-WebSocket-Extensions: ")
		bw.WriteString(extensions)
		bw.WriteString("\r\n")
	}

	bw.WriteString("\r\n")

	return bw.Flush()
}

func splitHeader(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if v := trimSpace(s[start:i]); v != "" {
				out = append(out, v)
			}
			start = i + 1
		}
	}
	if v := trimSpace(s[start:]); v != "" {
		out = append(out, v)
	}

	return out
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}

func headerContains(h http.Header, key, token string) bool {
	for _, v := range h[http.CanonicalHeaderKey(key)] {
		if asciiEqualFold(v, token) {
			return true
		}
	}

	return false
}

func asciiEqualFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		sr, tr := s[i], t[i]
		if sr == tr {
			continue
		}

		if 'A' <= sr && sr <= 'Z' {
			sr += 'a' - 'A'
		}
		if 'A' <= tr && tr <= 'Z' {
			tr += 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}

	return true
}

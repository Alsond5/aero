// websocket/errors.go

package websocket

import (
	"errors"
	"fmt"
)

var (
	ErrBadMethod              = errors.New("websocket: request method must be GET")
	ErrBadProtocol            = errors.New("websocket: HTTP version must be at least 1.1")
	ErrBadHost                = errors.New("websocket: missing Host header")
	ErrBadUpgrade             = errors.New("websocket: missing or invalid Upgrade header")
	ErrBadConnection          = errors.New("websocket: missing or invalid Connection header")
	ErrBadSecKey              = errors.New("websocket: missing or invalid Sec-WebSocket-Key")
	ErrBadSecVersion          = errors.New("websocket: missing Sec-WebSocket-Version")
	ErrHeaderLengthUnexpected = errors.New("websocket: header length is too large")
	ErrConnClosed             = errors.New("websocket: connection closed")
	ErrFrameTooLarge          = errors.New("websocket: frame too large")
	ErrBadHandshake           = errors.New("websocket: bad handshake")
	ErrNotWebSocket           = errors.New("websocket: request is not a websocket upgrade")
	ErrBadWebSocketKey        = errors.New("websocket: missing or invalid Sec-WebSocket-Key")
	ErrBadWebSocketVersion    = errors.New("websocket: client must use version 13")
	ErrBadWebSocketKeyLen     = errors.New("websocket: invalid Sec-WebSocket-Key length")
	ErrForbiddenOrigin        = errors.New("websocket: origin not allowed")
	ErrHijackNotSupport       = errors.New("websocket: response does not support hijacking")
	ErrUpgradeRequired        = errors.New("websocket: unsupported Sec-WebSocket-Version, only 13 is supported")
)

// CloseError is returned when the remote peer sends a close frame.
// It carries the status code and optional reason string.
type CloseError struct {
	Code   uint16
	Reason string
}

// NewCloseError creates a [CloseError] with the given status code and reason.
func NewCloseError(code CloseStatusCode, reason string) *CloseError {
	return &CloseError{
		Code:   code.Value(),
		Reason: reason,
	}
}

// Error implements the error interface, returning a human-readable
// description of the close event.
func (e *CloseError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("websocket: close %d", e.Code)
	}

	return fmt.Sprintf("websocket: close %d %s", e.Code, e.Reason)
}

// CloseStatusCode represents a WebSocket close status code as defined in
// RFC 6455 §7.4.
type CloseStatusCode uint16

const (
	CloseNormalClosure      CloseStatusCode = 1000
	CloseGoingAway          CloseStatusCode = 1001
	CloseProtocolError      CloseStatusCode = 1002
	CloseUnsupportedData    CloseStatusCode = 1003
	CloseNoStatusReceived   CloseStatusCode = 1005
	CloseAbnormalClosure    CloseStatusCode = 1006
	CloseInvalidPayload     CloseStatusCode = 1007
	ClosePolicyViolation    CloseStatusCode = 1008
	CloseMessageTooBig      CloseStatusCode = 1009
	CloseMandatoryExtension CloseStatusCode = 1010
	CloseInternalServerErr  CloseStatusCode = 1011
	CloseServiceRestart     CloseStatusCode = 1012
	CloseTryAgainLater      CloseStatusCode = 1013
	CloseTLSHandshake       CloseStatusCode = 1015
)

// Value returns the numeric uint16 representation of the status code,
// suitable for writing into a close frame payload.
func (s CloseStatusCode) Value() uint16 {
	return uint16(s)
}

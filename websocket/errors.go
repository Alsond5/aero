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

type CloseError struct {
	Code   uint16
	Reason string
}

func NewCloseError(code CloseStaatusCode, reason string) *CloseError {
	return &CloseError{
		Code:   code.Value(),
		Reason: reason,
	}
}

func (e *CloseError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("websocket: close %d", e.Code)
	}

	return fmt.Sprintf("websocket: close %d %s", e.Code, e.Reason)
}

type CloseStaatusCode uint16

const (
	CloseNormalClosure      CloseStaatusCode = 1000
	CloseGoingAway          CloseStaatusCode = 1001
	CloseProtocolError      CloseStaatusCode = 1002
	CloseUnsupportedData    CloseStaatusCode = 1003
	CloseNoStatusReceived   CloseStaatusCode = 1005
	CloseAbnormalClosure    CloseStaatusCode = 1006
	CloseInvalidPayload     CloseStaatusCode = 1007
	ClosePolicyViolation    CloseStaatusCode = 1008
	CloseMessageTooBig      CloseStaatusCode = 1009
	CloseMandatoryExtension CloseStaatusCode = 1010
	CloseInternalServerErr  CloseStaatusCode = 1011
	CloseServiceRestart     CloseStaatusCode = 1012
	CloseTryAgainLater      CloseStaatusCode = 1013
	CloseTLSHandshake       CloseStaatusCode = 1015
)

func (s CloseStaatusCode) Value() uint16 {
	return uint16(s)
}

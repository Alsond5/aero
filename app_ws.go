package aero

import (
	"slices"
	"sync"
	"time"

	"github.com/Alsond5/aero/websocket"
)

const bufSize = 4 << 10

var connPool = sync.Pool{
	New: func() any {
		b := make(buffer, 0, bufSize)

		return &WSConn{
			buf:    &b,
			locals: make(map[string]any),
		}
	},
}

// WebSocketHandler is the function signature for WebSocket connection handlers.
// The connection is automatically closed when the handler returns.
//
//	aero.WebSocket(func(ws *aero.WSConn) {
//		for {
//			mt, msg, err := ws.ReadMessage()
//			if err != nil {
//				break
//			}
//			ws.WriteMessage(mt, msg)
//		}
//	})
type WebSocketHandler func(*WSConn)

// WSConfig holds configuration options for a WebSocket endpoint.
// All fields are optional; unset fields fall back to sensible defaults.
type WSConfig struct {
	// Subprotocols is the list of supported WebSocket subprotocols.
	// The server negotiates the best match with the client's
	// Sec-WebSocket-Protocol header.
	Subprotocols []string

	// WriteTimeout is the maximum duration allowed for a single write
	// operation. A zero value means no timeout.
	WriteTimeout time.Duration

	// Origins is the list of allowed Origin header values for the handshake.
	// Requests with an origin not in this list are rejected with 403.
	// If empty, all origins are permitted.
	Origins []string

	// AllowEmptyOrigin permits WebSocket upgrade requests that carry no
	// Origin header. Useful for non-browser clients. Default: false.
	AllowEmptyOrigin bool

	// MaxMessageSize is the maximum allowed incoming message size in bytes.
	// Messages exceeding this limit cause the connection to be closed.
	// A zero value means no limit.
	MaxMessageSize uint64
}

func defaultWSConfig() WSConfig {
	return WSConfig{
		MaxMessageSize:   1 << 20,
		Origins:          []string{"*"},
		AllowEmptyOrigin: true,
	}
}

func setConfig(dst, src *WSConfig) {
	if src.MaxMessageSize > 0 {
		dst.MaxMessageSize = src.MaxMessageSize
	}
}

// WebSocket returns a [HandlerFunc] that upgrades the HTTP connection to
// WebSocket and calls fn with the established [WSConn]. The connection is
// closed automatically when fn returns. An optional [WSConfig] can be
// provided to configure subprotocols, origin policy, and message limits.
//
//	app.GET("/ws", aero.WebSocket(func(ws *aero.WSConn) {
//		mt, msg, err := ws.ReadMessage()
//		if err != nil {
//			return
//		}
//		ws.WriteMessage(mt, msg)
//	}))
func WebSocket(fn WebSocketHandler, config ...WSConfig) HandlerFunc {
	cfg := defaultWSConfig()
	if len(config) > 0 {
		setConfig(&cfg, &config[0])
	}

	hasWildcard := slices.Contains(cfg.Origins, "*")

	maxMessageSize := uint64(1 << 20)
	if cfg.MaxMessageSize > 0 {
		maxMessageSize = cfg.MaxMessageSize
	}

	var upgrader = websocket.Upgrader{
		Subprotocols: cfg.Subprotocols,
		WriteTimeout: cfg.WriteTimeout,
		CheckOrigin: func(origin string) bool {
			if len(cfg.Origins) == 1 && cfg.Origins[0] == "*" {
				return true
			}

			if origin == "" {
				return hasWildcard || cfg.AllowEmptyOrigin
			}

			return slices.Contains(cfg.Origins, origin)
		},
	}

	return func(c *Ctx) error {
		conn, err := upgrader.Upgrade(c.w, c.r)
		if err != nil {
			return err
		}

		c.isHijacked = true
		c.app.pool.Put(c)

		ws := acquireConn(conn)
		defer func() {
			ws.Close() //nolint:errcheck
			releaseConn(ws)
		}()

		ws.maxMessageSize = maxMessageSize

		fn(ws)
		return nil
	}
}

func acquireConn(conn *websocket.Conn) *WSConn {
	ws := connPool.Get().(*WSConn)
	ws.conn = conn

	return ws
}

func releaseConn(ws *WSConn) {
	clear(ws.locals)

	connPool.Put(ws)
}

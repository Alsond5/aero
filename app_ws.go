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

type WebSocketHandler func(*WSConn)

type WSConfig struct {
	Subprotocols     []string
	WriteTimeout     time.Duration
	Origins          []string
	AllowEmptyOrigin bool
	MaxMessageSize   uint64
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

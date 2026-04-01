package aero

import "github.com/Alsond5/aero/websocket"

var defaultUpgrader = &websocket.Upgrader{}

type WSHandlerFunc func(ws *Websocket)

type WSConfig struct {
	MaxMessageSize uint64
}

func NewWebsocket(handler WSHandlerFunc, config WSConfig) HandlerFunc {
	return NewWebsocketWithUpgrader(handler, defaultUpgrader, config)
}

func NewWebsocketWithUpgrader(handler WSHandlerFunc, upgrader *websocket.Upgrader, config WSConfig) HandlerFunc {
	return func(c *Ctx) error {
		conn, err := upgrader.Upgrade(c.w, c.r)
		if err != nil {
			return err
		}

		c.isHijacked = true
		c.app.pool.Put(c)

		ws := wsPool.Get().(*Websocket)
		ws.conn = conn
		ws.maxMessageSize = config.MaxMessageSize
		if ws.maxMessageSize == 0 {
			ws.maxMessageSize = 8 * 1024 * 1024
		}

		defer func() {
			ws.Close()
			ws.reset()

			wsPool.Put(ws)
		}()

		handler(ws)
		return nil
	}
}

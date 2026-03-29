package aero

import (
	"net/http"
	"net/url"
)

type Ctx struct {
	Req
	Res

	app         *App
	route       *Route
	r           *http.Request
	w           http.ResponseWriter
	middlewares []HandlerFunc
	index       int

	basePath    string
	path        string
	params      ParamValues
	paramsCount int
	query       url.Values
	status      int
	size        int64

	written     bool
	queryParsed bool
	formParsed  bool
}

func (c *Ctx) Next() error {
	if c.route == nil || c.index >= c.route.total {
		return nil
	}

	var h HandlerFunc
	if c.index < c.route.mc {
		h = c.middlewares[c.index]
	} else {
		h = c.route.handlers[c.index-c.route.mc]
	}

	c.index++
	return h(c)
}

func (c *Ctx) reset() {
	c.app = nil
	c.route = nil
	c.r = nil
	c.w = nil

	c.middlewares = c.middlewares[:0]
	c.index = 0

	c.basePath = ""
	c.path = ""
	c.params = c.params[:0]
	c.query = nil
	c.status = 200
	c.size = 0

	c.written = false
	c.queryParsed = false
	c.formParsed = false
}

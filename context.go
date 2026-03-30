package aero

import (
	"net/http"
	"net/url"
)

type Ctx struct {
	Req
	Res
	app         *App
	route       *route
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
	if c.index < c.route.middlewareCount {
		h = c.middlewares[c.index]
	} else {
		h = c.route.handlers[c.index-c.route.middlewareCount]
	}

	c.index++

	return h(c)
}

func (c *Ctx) reset(w http.ResponseWriter, r *http.Request) {
	c.route = nil
	c.w = w
	c.r = r

	c.index = 0

	c.basePath = ""
	c.path = ""
	c.paramsCount = 0
	c.query = nil
	c.status = 200
	c.size = 0

	c.written = false
	c.queryParsed = false
	c.formParsed = false
}

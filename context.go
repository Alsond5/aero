package aero

import (
	"net/http"
	"net/url"
)

// Ctx is the request context passed to every [HandlerFunc]. It embeds [Req]
// and [Res] directly, so request reading and response writing methods are
// accessible without any extra field access.
//
//	app.GET("/hello", func(c *aero.Ctx) error {
//		name := c.Req.Param("name")
//		return c.Res.SendString("Hello, " + name)
//	})
type Ctx struct {
	// Req provides access to all incoming request data: headers, params,
	// query string, body, cookies, and more.
	Req

	// Res provides methods for building and sending the HTTP response:
	// status codes, headers, JSON, files, redirects, and more.
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
	isHijacked  bool
}

// Next executes the next handler or middleware in the chain for the current
// route. It must be called inside a middleware to pass control forward.
// Calling Next after the chain is exhausted is a no-op and returns nil.
//
//	func logger(c *aero.Ctx) error {
//		start := time.Now()
//		err := c.Next()
//		log.Printf("%s %s — %v", c.Req.Method(), c.Req.Path(), time.Since(start))
//		return err
//	}
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

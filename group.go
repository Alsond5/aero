package aero

import "net/http"

type Group struct {
	app         *App
	middlewares []HandlerFunc
	prefix      string
}

func (g *Group) Use(middleware ...HandlerFunc) {
	g.middlewares = append(g.middlewares, middleware...)
}

func (g *Group) GET(path string, handlers ...HandlerFunc) {
	g.add(http.MethodGet, path, handlers)
}

func (g *Group) POST(path string, handlers ...HandlerFunc) {
	g.add(http.MethodPost, path, handlers)
}

func (g *Group) PUT(path string, handlers ...HandlerFunc) {
	g.add(http.MethodPut, path, handlers)
}

func (g *Group) PATCH(path string, handlers ...HandlerFunc) {
	g.add(http.MethodPatch, path, handlers)
}

func (g *Group) DELETE(path string, handlers ...HandlerFunc) {
	g.add(http.MethodDelete, path, handlers)
}

func (g *Group) HEAD(path string, handlers ...HandlerFunc) {
	g.add(http.MethodHead, path, handlers)
}

func (g *Group) OPTIONS(path string, handlers ...HandlerFunc) {
	g.add(http.MethodOptions, path, handlers)
}

func (g *Group) Group(prefix string, m ...HandlerFunc) (group *Group) {
	group = &Group{
		prefix:      prefix,
		app:         g.app,
		middlewares: make([]HandlerFunc, 0, len(m)),
	}
	group.Use(m...)
	return
}

func (g *Group) add(method, path string, handlers []HandlerFunc) {
	path = g.prefix + path

	if len(g.middlewares) > 0 {
		m := make([]HandlerFunc, 0, len(g.middlewares)+len(handlers))
		m = append(m, g.middlewares...)
		m = append(m, handlers...)

		handlers = m
	}

	g.app.add(method, path, handlers)
}

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

func (g *Group) GET(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodGet, path, h, m)
}

func (g *Group) POST(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodPost, path, h, m)
}

func (g *Group) PUT(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodPut, path, h, m)
}

func (g *Group) PATCH(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodPatch, path, h, m)
}

func (g *Group) DELETE(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodDelete, path, h, m)
}

func (g *Group) HEAD(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodHead, path, h, m)
}

func (g *Group) OPTIONS(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodOptions, path, h, m)
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

func (g *Group) add(method, path string, h HandlerFunc, m []HandlerFunc) {
	path = g.prefix + path

	totalCapacity := len(g.middlewares) + len(m) + 1
	handlers := make([]HandlerFunc, 0, totalCapacity)

	handlers = append(handlers, g.middlewares...)
	handlers = append(handlers, m...)
	handlers = append(handlers, h)

	g.app.add(method, path, handlers)
}

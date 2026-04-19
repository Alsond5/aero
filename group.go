package aero

import "net/http"

// Group represents a set of routes sharing a common path prefix
// and an optional set of group-level middlewares.
// Create a group via [App.Group] or [Group.Group].
type Group struct {
	app         *App
	middlewares []HandlerFunc
	prefix      string
}

// Use registers one or more middlewares scoped to this group.
// These middlewares run after global middlewares and before the route handler.
//
//	admin := app.Group("/admin")
//	admin.Use(adminAuthMiddleware)
func (g *Group) Use(middleware ...HandlerFunc) {
	g.middlewares = append(g.middlewares, middleware...)
}

// GET registers a GET route under this group's prefix.
//
//	v1 := app.Group("/v1")
//	v1.GET("/ping", pingHandler)  // → GET /v1/ping
func (g *Group) GET(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodGet, path, h, m)
}

// POST registers a POST route under this group's prefix.
func (g *Group) POST(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodPost, path, h, m)
}

// PUT registers a PUT route under this group's prefix.
func (g *Group) PUT(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodPut, path, h, m)
}

// PATCH registers a PATCH route under this group's prefix.
func (g *Group) PATCH(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodPatch, path, h, m)
}

// DELETE registers a DELETE route under this group's prefix.
func (g *Group) DELETE(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodDelete, path, h, m)
}

// HEAD registers a HEAD route under this group's prefix.
func (g *Group) HEAD(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodHead, path, h, m)
}

// OPTIONS registers an OPTIONS route under this group's prefix.
func (g *Group) OPTIONS(path string, h HandlerFunc, m ...HandlerFunc) {
	g.add(http.MethodOptions, path, h, m)
}

// Group creates a nested sub-group under the current group's prefix.
// The sub-group inherits the parent's middlewares and prepends its own prefix.
//
//	api := app.Group("/api")
//	v2 := api.Group("/v2", versionMiddleware)  // → prefix: /api/v2
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

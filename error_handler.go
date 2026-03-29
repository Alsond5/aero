package aero

import "net/http"

type ErrorHandler func(c *Ctx, err error)
type NotFoundHandler func(c *Ctx)
type MethodNotAllowedHandler func(allowed string, c *Ctx)

func defaultErrorHandler(c *Ctx, err error) {
	if !c.written {
		http.Error(c.w, err.Error(), http.StatusInternalServerError)
	}
}

func defaultNotFoundHandler(c *Ctx) {
	http.NotFound(c.w, c.r)
}

func defaultMethodNotAllowedHandler(allowed string, c *Ctx) {
	c.w.Header().Set("Allow", allowed)
	http.Error(c.w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

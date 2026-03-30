package aero

import "strings"

type Param struct {
	Key   string
	Value string
}

type ParamValues [maxParamCount]Param

type route struct {
	method   string
	path     string
	handlers []HandlerFunc

	middlewareCount int
	total           int
}

type endpoint struct {
	routes  [mCount]*route
	allowed methodBit
}

func newEndpoint() *endpoint {
	return &endpoint{}
}

func (e *endpoint) setRoute(mi int, route *route) {
	e.routes[mi] = route
	e.allowed |= methodBits[mi]

	if mi == mGET {
		e.allowed |= methodBitHEAD
	}
}

func (e *endpoint) getRoute(mi int) *route {
	return e.routes[mi]
}

func (e *endpoint) isAllowed(mi int) bool {
	return e.allowed&methodBits[mi] != 0
}

func (e *endpoint) allowedMethods() string {
	var b strings.Builder

	first := true
	for mi := range mCount {
		if !e.isAllowed(mi) {
			continue
		}

		if !first {
			b.WriteString(", ")

		}

		b.WriteString(methodString(mi))
		first = false
	}

	return b.String()
}

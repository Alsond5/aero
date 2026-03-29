package aero

import "strings"

const (
	mGET     = 0
	mPOST    = 1
	mPUT     = 2
	mPATCH   = 3
	mDELETE  = 4
	mHEAD    = 5
	mOPTIONS = 6
	mCount   = 7
)

func methodIndex(m string) int {
	switch m {
	case "GET":
		return mGET
	case "POST":
		return mPOST
	case "PUT":
		return mPUT
	case "PATCH":
		return mPATCH
	case "DELETE":
		return mDELETE
	case "HEAD":
		return mHEAD
	case "OPTIONS":
		return mOPTIONS
	}

	return -1
}

func methodString(mi int) string {
	switch mi {
	case mGET:
		return "GET"
	case mPOST:
		return "POST"
	case mPUT:
		return "PUT"
	case mPATCH:
		return "PATCH"
	case mDELETE:
		return "DELETE"
	case mHEAD:
		return "HEAD"
	case mOPTIONS:
		return "OPTIONS"
	}

	return ""
}

type methodBit uint16

const (
	methodBitGET    methodBit = 1 << 0
	methodBitPOST   methodBit = 1 << 1
	methodBitPUT    methodBit = 1 << 2
	methodBitPATCH  methodBit = 1 << 3
	methodBitDELETE methodBit = 1 << 4
	methodBitHEAD   methodBit = 1 << 5
)

var methodBits = [mCount]methodBit{
	mGET:    methodBitGET,
	mPOST:   methodBitPOST,
	mPUT:    methodBitPUT,
	mPATCH:  methodBitPATCH,
	mDELETE: methodBitDELETE,
	mHEAD:   methodBitHEAD,
}

type ParamValues []Param

type Param struct {
	Key   string
	Value string
}

type staticTable map[string]*endpoint

type route struct {
	method   string
	path     string
	handlers []HandlerFunc
	params   []string

	middlewareCount int
	total           int

	isStatic   bool
	isWildcard bool
	isRoot     bool
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

type Router struct {
	tree   *segmentTrie
	static staticTable
}

func NewRouter() *Router {
	r := &Router{
		tree:   newSegmentTrie(),
		static: make(staticTable, 10),
	}

	return r
}

func (r *Router) register(method, path string, handlers []HandlerFunc, middlewareCount int) {
	mi := methodIndex(method)
	if mi == -1 {
		panic("unsupported HTTP method: " + method)
	}

	dynamic, labels := parsePath(path)

	route := &route{
		method:          method,
		path:            path,
		handlers:        handlers,
		params:          labels,
		isStatic:        !dynamic,
		isRoot:          path == "/",
		middlewareCount: middlewareCount,
		total:           middlewareCount + len(handlers),
	}

	if dynamic {
		r.tree.Insert(path, mi, route)
	} else {
		ep, ok := r.static[path]
		if !ok {
			ep = newEndpoint()
			r.static[path] = ep
		}

		ep.setRoute(mi, route)
	}
}

func (r *Router) match(path string, params *ParamValues, paramsCount *int) *endpoint {
	if ep, ok := r.static[path]; ok {
		return ep
	}

	return r.tree.Search(path, params, paramsCount)
}

func parsePath(path string) (bool, []string) {
	params := make([]string, 0, 2)
	dynamic := false

	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '*':
			dynamic = true
			return dynamic, params
		case ':':
			dynamic = true
			i++
			end := i

			for end < len(path) && path[end] != '/' {
				end++
			}

			params = append(params, path[i:end])
			i = end - 1
		}
	}

	return dynamic, params
}

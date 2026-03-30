package aero

type staticTable map[string]*endpoint

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

	route := &route{
		method:          method,
		path:            path,
		handlers:        handlers,
		middlewareCount: middlewareCount,
		total:           middlewareCount + len(handlers),
	}

	dynamic, paramCount := analyzePath(path)
	if paramCount > maxParamCount {
		panic("too many params in route: max is " + maxParamCountStr)
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

func analyzePath(path string) (bool, int) {
	paramCount := 0
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case ':':
			paramCount++
		case '*':
			return true, paramCount
		}
	}

	return paramCount > 0, paramCount
}

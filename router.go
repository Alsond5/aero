package aero

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

type staticTable [mCount]map[string]*Route

type ParamValues []Param

type Param struct {
	Key   string
	Value string
}

type Route struct {
	method     string
	path       string
	params     []string
	handlers   []HandlerFunc
	mc         int
	total      int
	isStatic   bool
	isWildcard bool
	isRoot     bool
}

type Router struct {
	static  staticTable
	dynamic [mCount]*SegmentTrie
}

func newRouter() *Router {
	r := &Router{}

	for i := range mCount {
		r.static[i] = make(map[string]*Route, 4)
		r.dynamic[i] = newARTTree()
	}

	return r
}

func (r *Router) register(method, path string, handlers []HandlerFunc, middlewareCount int) {
	mi := methodIndex(method)
	if mi == -1 {
		panic("unsupported HTTP method: " + method)
	}

	isDynamic, paramNames := parsePath(path)

	route := &Route{
		method:   method,
		path:     path,
		handlers: handlers,
		mc:       middlewareCount,
		total:    middlewareCount + len(handlers),
		params:   paramNames,
		isStatic: !isDynamic,
		isRoot:   path == "/",
	}

	if isDynamic {
		r.dynamic[mi].Insert(path, route)
	} else {
		r.static[mi][path] = route
	}
}

func (r *Router) match(method, path string, params *ParamValues, paramsCount *int) *Route {
	mi := methodIndex(method)
	if mi == -1 {
		return nil
	}

	if route, ok := r.static[mi][path]; ok {
		return route
	}

	return r.dynamic[mi].Search(path, params, paramsCount)
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

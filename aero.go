package aero

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

// MB is a convenience constant representing one megabyte (1 << 20 bytes).
// Used as a base unit for body size limits in [Config].
const (
	MB int64 = 1 << 20
)

var _ http.Handler = (*App)(nil)

// HandlerFunc is the core handler type in Aero. Every route handler and
// middleware must match this signature. Returning a non-nil error passes
// it to the application's [ErrorHandler].
//
//	app.GET("/hello", func(c *aero.Ctx) error {
//		return c.Res.SendString("Hello, World!")
//	})
type HandlerFunc func(c *Ctx) error

// Config holds application-level configuration options.
// All fields are optional; unset fields fall back to the defaults
// defined in [defaultConfig].
//
//	app := aero.New(aero.Config{
//		TrustProxy:  true,
//		MaxBodySize: 8 * aero.MB,
//	})
type Config struct {
	// TrustProxy enables parsing of proxy-forwarded headers such as
	// X-Forwarded-For and X-Forwarded-Proto when determining the
	// client IP and protocol. Enable only if Aero sits behind a
	// trusted reverse proxy. Default: false.
	TrustProxy bool

	// SubdomainOffset controls how many dot-separated segments are
	// stripped from the right of the hostname when extracting subdomains
	// via [Req.Subdomains]. For example, an offset of 2 on "a.b.example.com"
	// yields ["a", "b"]. Default: 2.
	SubdomainOffset int

	// MaxBodySize is the maximum allowed size in bytes for a request body.
	// Requests exceeding this limit are rejected before the body is read.
	// Default: 4 MB.
	MaxBodySize int64

	// MaxMultipartMemory is the maximum amount of memory in bytes used
	// when parsing multipart/form-data requests. Parts exceeding this
	// limit are stored in temporary files. Default: 32 MB.
	MaxMultipartMemory int64
}

func defaultConfig() Config {
	return Config{
		TrustProxy:         false,
		SubdomainOffset:    2,
		MaxBodySize:        4 * MB,
		MaxMultipartMemory: 32 * MB,
	}
}

// App is the core Aero application instance. It holds the router,
// middleware chain, configuration, and lifecycle hooks.
// Create a new instance with [New].
//
// App implements [http.Handler], so it can be passed directly to
// any standard Go HTTP server.
type App struct {
	config      Config
	pool        sync.Pool
	middlewares []HandlerFunc
	onShutdown  []func()
	routeCount  atomic.Uint32
	mu          sync.RWMutex
	router      *router
	validator   Validator

	// NotFoundHandler is called when no route matches the request path.
	// Defaults to a 404 JSON response if not set.
	NotFoundHandler NotFoundHandler

	// MethodNotAllowedHandler is called when a route exists for the
	// requested path but not for the requested HTTP method.
	// Defaults to a 405 response with an Allow header if not set.
	MethodNotAllowedHandler MethodNotAllowedHandler

	// ErrorHandler is called whenever a HandlerFunc returns a non-nil error.
	// Defaults to a 500 JSON response if not set.
	ErrorHandler ErrorHandler

	// OptionsHandler is called for OPTIONS requests that do not have
	// an explicitly registered OPTIONS route.
	// Defaults to a 204 response with an Allow header if not set.
	OptionsHandler OptionsHandler
}

// New creates and returns a new Aero application instance.
// An optional [Config] can be provided to override defaults.
//
//	app := aero.New()
//	app := aero.New(aero.Config{ ... })
func New(config ...Config) *App {
	cfg := defaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	app := &App{
		config: cfg,
		router: newRouter(),
	}

	app.pool = sync.Pool{
		New: func() any {
			c := &Ctx{
				app:    app,
				status: 200,
			}
			c.Req.c = c
			c.Res.c = c

			return c
		},
	}

	app.NotFoundHandler = defaultNotFoundHandler
	app.MethodNotAllowedHandler = defaultMethodNotAllowedHandler
	app.ErrorHandler = defaultErrorHandler
	app.OptionsHandler = defaultOptionsHandler

	return app
}

// GET registers a route for HTTP GET requests on the given path.
// Route-level middlewares can be appended after the handler.
//
//	app.GET("/users", listUsers)
//	app.GET("/users/:id", getUser, authMiddleware)
func (a *App) GET(path string, h HandlerFunc, m ...HandlerFunc) {
	a.add(http.MethodGet, path, applyMiddlewares(h, m))
}

// POST registers a route for HTTP POST requests on the given path.
func (a *App) POST(path string, h HandlerFunc, m ...HandlerFunc) {
	a.add(http.MethodPost, path, applyMiddlewares(h, m))
}

// PUT registers a route for HTTP PUT requests on the given path.
func (a *App) PUT(path string, h HandlerFunc, m ...HandlerFunc) {
	a.add(http.MethodPut, path, applyMiddlewares(h, m))
}

// PATCH registers a route for HTTP PATCH requests on the given path.
func (a *App) PATCH(path string, h HandlerFunc, m ...HandlerFunc) {
	a.add(http.MethodPatch, path, applyMiddlewares(h, m))
}

// DELETE registers a route for HTTP DELETE requests on the given path.
func (a *App) DELETE(path string, h HandlerFunc, m ...HandlerFunc) {
	a.add(http.MethodDelete, path, applyMiddlewares(h, m))
}

// HEAD registers a route for HTTP HEAD requests on the given path.
func (a *App) HEAD(path string, h HandlerFunc, m ...HandlerFunc) {
	a.add(http.MethodHead, path, applyMiddlewares(h, m))
}

// OPTIONS registers a route for HTTP OPTIONS requests on the given path.
func (a *App) OPTIONS(path string, h HandlerFunc, m ...HandlerFunc) {
	a.add(http.MethodOptions, path, applyMiddlewares(h, m))
}

// TRACE registers a route for HTTP TRACE requests on the given path.
func (a *App) TRACE(path string, h HandlerFunc, m ...HandlerFunc) {
	a.add(http.MethodTrace, path, applyMiddlewares(h, m))
}

// Use registers one or more global middlewares that run on every request,
// regardless of the route. Middlewares are executed in the order they are added.
//
//	app.Use(logger, recover, cors)
func (a *App) Use(handlers ...HandlerFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.middlewares = append(a.middlewares, handlers...)
}

// Group creates a new route group with the given path prefix and optional
// group-level middlewares. Routes registered on the group inherit the prefix
// and its middlewares.
//
//	api := app.Group("/api", authMiddleware)
//	api.GET("/users", listUsers)  // → GET /api/users
func (a *App) Group(prefix string, m ...HandlerFunc) (g *Group) {
	g = &Group{
		prefix:      prefix,
		app:         a,
		middlewares: make([]HandlerFunc, 0, len(m)),
	}
	g.Use(m...)
	return
}

// SetValidator sets the validator used by [Req.Validate].
// The validator must implement the [Validator] interface.
func (a *App) SetValidator(v Validator) {
	a.validator = v
}

// ServeHTTP implements [http.Handler], enabling Aero to be used with
// any standard Go HTTP server or middleware.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := a.pool.Get().(*Ctx)
	defer func() {
		if !ctx.isHijacked {
			a.pool.Put(ctx)
		}
	}()

	ctx.reset(w, r)

	ep := a.router.match(r.URL.Path, &ctx.params, &ctx.paramsCount)
	if ep == nil {
		a.NotFoundHandler(ctx)
		return
	}

	route := ep.getRoute(methodIndex(r.Method))
	if route == nil && r.Method == http.MethodHead && ep.isAllowed(mGET) {
		route = ep.getRoute(mGET)
	}
	if route == nil {
		if r.Method == http.MethodOptions {
			err := a.OptionsHandler(ep.allowedMethods(), ctx)
			if err != nil {
				a.ErrorHandler(ctx, err)
			}

			return
		}

		a.MethodNotAllowedHandler(ep.allowedMethods(), ctx)
		return
	}

	ctx.route = route
	ctx.middlewares = a.middlewares[:route.middlewareCount]

	if err := ctx.Next(); err != nil {
		a.ErrorHandler(ctx, err)
	}
}

// Listen starts the HTTP server on the given address.
// The address format follows standard Go net/http conventions (e.g. ":8080").
//
//	app.Listen(":8080")
func (a *App) Listen(addr string) error {
	sc := ServerConfig{Addr: addr}
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	return sc.Start(ctx, a)
}

// ListenTLS starts an HTTPS server on the given address using the provided
// TLS certificate and key files.
//
//	app.ListenTLS(":443", "cert.pem", "key.pem")
func (a *App) ListenTLS(addr, cert, key string) error {
	sc := ServerConfig{
		Addr:    addr,
		TLSCert: cert,
		TLSKey:  key,
	}
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	return sc.StartTLS(ctx, a)
}

func (a *App) add(method, path string, handlers []HandlerFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.router.register(method, path, handlers, len(a.middlewares))
	a.routeCount.Add(1)
}

func applyMiddlewares(h HandlerFunc, m []HandlerFunc) []HandlerFunc {
	handlers := make([]HandlerFunc, 0, len(m)+1)

	handlers = append(handlers, m...)
	handlers = append(handlers, h)

	return handlers
}

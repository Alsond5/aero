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

const (
	MB int64 = 1 << 20
)

var _ http.Handler = (*App)(nil)

type HandlerFunc func(c *Ctx) error

type Config struct {
	TrustProxy         bool
	SubdomainOffset    int
	MaxBodySize        int64
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

type App struct {
	config      Config
	pool        sync.Pool
	middlewares []HandlerFunc
	onShutdown  []func()
	routeCount  atomic.Uint32
	mu          sync.RWMutex

	router *Router

	NotFoundHandler         NotFoundHandler
	MethodNotAllowedHandler MethodNotAllowedHandler
	ErrorHandler            ErrorHandler
	OptionsHandler          OptionsHandler
}

func New(config ...Config) *App {
	cfg := defaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	app := &App{
		config: cfg,
		router: NewRouter(),
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

func (a *App) GET(path string, handlers ...HandlerFunc) {
	a.add("GET", path, handlers)
}

func (a *App) POST(path string, handlers ...HandlerFunc) {
	a.add("POST", path, handlers)
}

func (a *App) PUT(path string, handlers ...HandlerFunc) {
	a.add("PUT", path, handlers)
}

func (a *App) PATCH(path string, handlers ...HandlerFunc) {
	a.add("PATCH", path, handlers)
}

func (a *App) DELETE(path string, handlers ...HandlerFunc) {
	a.add("DELETE", path, handlers)
}

func (a *App) HEAD(path string, handlers ...HandlerFunc) {
	a.add("HEAD", path, handlers)
}

func (a *App) OPTIONS(path string, handlers ...HandlerFunc) {
	a.add("OPTIONS", path, handlers)
}

func (a *App) Use(handlers ...HandlerFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.middlewares = append(a.middlewares, handlers...)
}

func (a *App) add(method, path string, handlers []HandlerFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.router.register(method, path, handlers, len(a.middlewares))
	a.routeCount.Add(1)
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := a.pool.Get().(*Ctx)
	defer a.pool.Put(ctx)

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
			a.OptionsHandler(ep.allowedMethods(), ctx)
			return
		}

		a.MethodNotAllowedHandler(ep.allowedMethods(), ctx)
		return
	}

	ctx.route = route
	if route.middlewareCount > 0 {
		ctx.middlewares = a.middlewares
	}

	if err := ctx.Next(); err != nil {
		a.ErrorHandler(ctx, err)
	}
}

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

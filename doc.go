// Package aero is a zero-dependency, high-performance HTTP web framework
// for Go. It is designed for minimal allocations
// on the hot path, making it suitable for latency-sensitive applications.
//
// # Overview
//
// Aero provides a fast trie-based router with support for named URL parameters,
// route groups, and a hybrid middleware chain model that allows both global and
// group/route-level middlewares. The framework is built entirely on top of the
// Go standard library with no external dependencies.
//
// Key characteristics:
//   - Zero external dependencies
//   - Trie-based router with zero allocations on matched routes
//   - Hybrid middleware chain: global ([App.Use]) and scoped ([Group.Use])
//   - First-class WebSocket support via a custom RFC 6455 implementation
//   - Built-in request binding, content negotiation, and SSE support
//   - Graceful shutdown via [ServerConfig]
//
// # Getting Started
//
// Create an application with [New], register routes, and start the server:
//
//	app := aero.New()
//
//	app.GET("/", func(c *aero.Ctx) error {
//		return c.Res.SendString("Hello, World!")
//	})
//
//	app.Listen(":8080")
//
// # Route Groups
//
// Routes can be grouped under a common prefix with shared middlewares:
//
//	api := app.Group("/api", authMiddleware)
//
//	api.GET("/users", listUsers)        // GET /api/users
//	api.POST("/users", createUser)      // POST /api/users
//
// # Middleware
//
// Middlewares are regular [HandlerFunc] functions that call [Ctx.Next] to
// pass control to the next handler in the chain:
//
//	app.Use(func(c *aero.Ctx) error {
//		start := time.Now()
//		err := c.Next()
//		log.Printf("%s %s — %v", c.Req.Method(), c.Req.Path(), time.Since(start))
//		return err
//	})
//
// # WebSocket
//
// WebSocket endpoints are registered as regular routes using [WebSocket]:
//
//	app.GET("/ws", aero.WebSocket(func(ws *aero.WSConn) {
//		for {
//			mt, msg, err := ws.ReadMessage()
//			if err != nil {
//				return
//			}
//			ws.WriteMessage(mt, msg)
//		}
//	}))
//
// # Request Binding
//
// Incoming request data can be bound directly into structs. Each field's
// struct tag determines the source (json, form, query, param, header):
//
//	type CreateUserReq struct {
//		OrgID string `param:"orgId"`
//		Token string `header:"X-Auth-Token"`
//		Name  string `json:"name"`
//	}
//
//	var req CreateUserReq
//	if err := c.Req.Bind(&req); err != nil {
//		return err
//	}
//
// # Graceful Shutdown
//
// For production use, [ServerConfig] provides context-aware lifecycle control:
//
//	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
//	defer cancel()
//
//	sc := aero.ServerConfig{
//		Addr:            ":8080",
//		GracefulTimeout: 10 * time.Second,
//	}
//
//	if err := sc.Start(ctx, app); err != nil {
//		log.Fatal(err)
//	}
package aero

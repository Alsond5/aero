# Aero

Blazing fast, zero-dependency, lightweight web framework for Go.

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-green)
![Zero Dependencies](https://img.shields.io/badge/dependencies-zero-brightgreen)

## Why Aero?

Most Go frameworks either sacrifice stdlib compatibility (Fiber/FastHTTP) 
or leave allocations on the table (Echo, Chi). Aero stays on net/http, 
achieves zero allocations through careful design, and ships with a 
production-ready WebSocket stack. No external dependencies required.

## Features

- Zero dependencies, pure stdlib, no transitive vulnerabilities
- Zero allocations in the hot path
- Faster than `net/http` 1.8x throughput, 60x less memory per request
- Auto HTTP2 support
- Hybrid routing with O(1) static map for exact paths, Segment Trie for dynamic ones
- Order-sensitive middleware
- WebSocket support (14% higher push throughput than Fiber v3)

## Guide

### Installation

```bash
go get github.com/Alsond5/aero
```

Requires Go 1.25+

### Using

```go
package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Alsond5/aero"
)

func main() {
	app := aero.New()

	app.Use(Logger)

	app.GET("/api/users/:id", func(c *aero.Ctx) error {
		id := c.Param("id")
		return c.JSON(map[string]string{"id": id})
	})

	app.GET("/ws", aero.WebSocket(func(ws *aero.WSConn) {
		ws.Locals("client", "id")
		fmt.Println(ws.Locals("client"))

		for {
			mt, msg, err := ws.ReadMessage()
			if err != nil {
				fmt.Println(err)
				break
			}

			ws.WriteMessage(mt, msg)
		}
	}))

	app.Listen(":8080")
}

func Logger(c *aero.Ctx) error {
	start := time.Now()
	err := c.Next()
	duration := time.Since(start)

	slog.Info("request",
		"method", c.Method(),
		"path", c.Path(),
		"ip", c.IP(),
		"duration", duration,
		"status", c.ResponseStatus(),
	)

	return err
}
```

## Benchmarks
 
**Machine:** Intel Core i7-10750H (unplugged, low-power mode)
**Go:** 1.25
**OS:** Fedora Linux, amd64
**Test:** 1 middleware + 2 GET routes (1 dynamic, 1 static) + 1 POST route with JSON body parse

### Standart

| Framework | ns/op  | B/op | allocs/op | req/s     |
|-----------|--------|------|-----------|-----------|
| **Aero**      | **603.8**  | **339**  | **3**         | **1,656,177** |
| Echo v5   | 772.7  | 336  | 4         | 1,294,163 |
| net/http  | 727.9  | 333  | 4         | 1,373,815 |
| Gin v1    | 955.2  | 368  | 5         | 1,046,901 |
| Chi v5    | 1442.7 | 808  | 6         | 693,144   |

### Routing

| Framework | ns/op  | B/op | allocs/op | req/s     |
|-----------|--------|------|-----------|-----------|
| **Aero**      | **194.8**  | **0**    | **0**         | **5,133,470** |
| Echo v5   | 232.6  | 16   | 1         | 4,299,226 |
| net/http  | 242.2  | 20   | 1         | 4,128,819 |
| Gin v1    | 277.6  | 52   | 2         | 3,602,305 |
| Chi v5    | 1109.8 | 548  | 4         | 901,063   |

### Parallel (12 goroutines)

| Framework | ns/op | B/op | allocs/op | req/s      |
|-----------|-------|------|-----------|------------|
| **Aero**      | **36.2**  | **0**    | **0**         | **27,624,309** |
| Echo v5   | 46.3  | 16   | 1         | 21,598,272 |
| net/http  | 60.2  | 20   | 1         | 16,611,295 |
| Gin v1    | 61.2  | 52   | 2         | 16,339,869 |
| Chi v5    | 179.7 | 548  | 4         | 5,564,830  |

### WebSocket (2000 concurrent connections)

**Test:** 2000 concurrent connections, 60s, 4 payload sizes (64B / 1KB / 4KB / 8KB), echo server

| Framework | msg/s | p95 RTT | p99 RTT | max RTT | Throughput |
|-----------|-------|---------|---------|---------|------------|
| **Aero** | **74,566** | **30ms** | **53ms** | **104ms** | **250 MB/s** |
| Fiber v3 | 62,978 | 37ms | 59ms | 178ms | 211 MB/s |

### WebSocket — Push Throughput (32 clients, 5s)

**Test:** Server pushes messages to 32 concurrent clients as fast as possible

| Framework | msg/s |
|-----------|-------|
| **Aero** | **5,772,032** |
| Fiber v3 | 5,056,723 |

## Design Decisions

Every feature is built on the Go standard library. No transitive vulnerabilities, no version conflicts, no go mod tidy surprises.

URL routing operates on path segments, not individual bytes. Radix Tree compression offers no benefit at segment granularity. A Segment Trie with a static map fast-path is simpler and equally fast.

Global middleware is stored once on the App. Routes store only an integer (middlewareCount). At dispatch time, the middleware slice and route handlers are indexed directly. No copying, no appending.

FastHTTP offers raw performance gains but breaks compatibility with the standard http.Handler ecosystem. Aero stays on net/http and achieves zero-alloc performance through careful design rather than a custom HTTP stack.

WebSocket frames are read with zero heap allocations. Buffers are pooled with borrow-on-demand to avoid per-connection overhead. Internally implements RFC 6455 from scratch using only the Go standard library no external dependencies.

## License

[MIT License](./LICENSE)

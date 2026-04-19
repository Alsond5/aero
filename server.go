package aero

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	readTimeout  = 30 * time.Second
	writeTimeout = 30 * time.Second
)

var ErrTLSCertRequired = errors.New("aero: TLS cert and key are required")

// ServerConfig holds the configuration for starting an Aero server
// with context-aware lifecycle management and graceful shutdown.
// Use [ServerConfig.Start] or [ServerConfig.StartTLS] instead of
// [App.Listen] when you need shutdown control via [context.Context].
//
//	sc := aero.ServerConfig{Addr: ":8080"}
//	if err := sc.Start(ctx, app); err != nil {
//		log.Fatal(err)
//	}
type ServerConfig struct {
	// Addr is the TCP address to listen on (e.g. ":8080", "0.0.0.0:443").
	// Ignored if Listener is provided.
	Addr string

	// Listener is an optional pre-configured [net.Listener]. When set,
	// Addr is ignored and the server accepts connections from this listener.
	// Useful for testing or socket activation scenarios.
	Listener net.Listener

	// TLSCert is the path to the TLS certificate file (PEM encoded).
	// Required for [ServerConfig.StartTLS].
	TLSCert string

	// TLSKey is the path to the TLS private key file (PEM encoded).
	// Required for [ServerConfig.StartTLS].
	TLSKey string

	// GracefulTimeout is the maximum duration to wait for in-flight requests
	// to complete during shutdown. If zero, the server waits indefinitely
	// until ctx is cancelled.
	GracefulTimeout time.Duration

	// OnShutdownError is an optional callback invoked if the graceful shutdown
	// itself returns an error (e.g. timeout exceeded). If nil, shutdown errors
	// are silently discarded.
	OnShutdownError func(err error)
}

// Start starts the HTTP server and blocks until ctx is cancelled,
// at which point it initiates a graceful shutdown — waiting for
// in-flight requests to complete before returning.
func (sc ServerConfig) Start(ctx context.Context, app *App) error {
	return sc.start(ctx, app, false)
}

// StartTLS starts an HTTPS server using the certificate and key files
// configured in ServerConfig. Blocks and shuts down gracefully on
// ctx cancellation, same as [ServerConfig.Start].
func (sc ServerConfig) StartTLS(ctx context.Context, app *App) error {
	return sc.start(ctx, app, true)
}

func (sc ServerConfig) start(ctx context.Context, app *App, tls bool) error {
	server := &http.Server{
		Handler:      app,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	listener := sc.Listener
	if listener == nil {
		var err error

		listener, err = net.Listen("tcp", sc.Addr)
		if err != nil {
			return err
		}
	}

	printBanner(listener.Addr().String())

	wg := sync.WaitGroup{}
	defer wg.Wait()

	gCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Go(func() {
		gracefulShutdown(gCtx, sc, server, app)
	})

	if !tls {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			return err
		}

		return nil
	}

	if sc.TLSCert == "" || sc.TLSKey == "" {
		return ErrTLSCertRequired
	}
	if err := server.ServeTLS(listener, sc.TLSCert, sc.TLSKey); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func gracefulShutdown(ctx context.Context, sc ServerConfig, server *http.Server, app *App) {
	<-ctx.Done()

	timeout := sc.GracefulTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, fn := range app.onShutdown {
		fn()
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		if sc.OnShutdownError != nil {
			sc.OnShutdownError(err)
			return
		}
	}
}

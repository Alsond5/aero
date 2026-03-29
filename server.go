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

type ServerConfig struct {
	Addr            string
	Listener        net.Listener
	TLSCert         string
	TLSKey          string
	GracefulTimeout time.Duration
	OnShutdownError func(err error)
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

func (sc ServerConfig) Start(ctx context.Context, app *App) error {
	return sc.start(ctx, app, false)
}

func (sc ServerConfig) StartTLS(ctx context.Context, app *App) error {
	return sc.start(ctx, app, true)
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

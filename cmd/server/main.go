// SPDX-FileCopyrightText: 2025 Fatih Ka. <xybydy@gmail.com>
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/xybydy/tsdproxy/internal/consts"

	"github.com/xybydy/tsdproxy/internal/config"
	"github.com/xybydy/tsdproxy/internal/core"
	"github.com/xybydy/tsdproxy/internal/dashboard"
	pm "github.com/xybydy/tsdproxy/internal/proxymanager"
)

type WebApp struct {
	Log          zerolog.Logger
	HTTP         *core.HTTPServer
	Health       *core.Health
	ProxyManager *pm.ProxyManager
	Dashboard    *dashboard.Dashboard
	cancel       context.CancelFunc
}

func InitializeApp() (*WebApp, error) {
	err := config.InitializeConfig()
	if err != nil {
		return nil, err
	}
	logger := core.NewLog()

	httpServer := core.NewHTTPServer(logger)
	httpServer.Use(core.SessionMiddleware)

	health := core.NewHealthHandler(httpServer, logger)

	// Start ProxyManager
	//
	proxymanager := pm.NewProxyManager(logger)

	// init Dashboard
	//
	dash := dashboard.NewDashboard(httpServer, logger, proxymanager)

	webApp := &WebApp{
		Log:          logger,
		HTTP:         httpServer,
		Health:       health,
		ProxyManager: proxymanager,
		Dashboard:    dash,
	}
	return webApp, nil
}

func main() {
	fmt.Println("Initializing server")
	fmt.Println("Version", core.GetVersion())

	app, err := InitializeApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the application with the context
	app.Start(ctx)
	defer app.Stop()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, consts.SignalChannelBufferSize)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-quit:
		app.Log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel() // Cancel context to stop all goroutines
	case <-ctx.Done():
		app.Log.Info().Msg("Context canceled")
	}
}

func (app *WebApp) Start(ctx context.Context) {
	app.Log.Info().
		Str("Version", core.GetVersion()).Msg("Starting server")

	// Store cancel function for later use
	ctx, cancel := context.WithCancel(ctx)
	app.cancel = cancel

	// Start the webserver
	go func() {
		app.Log.Info().Msg("Initializing WebServer")

		srv := http.Server{
			Addr:              fmt.Sprintf("%s:%d", config.Config.HTTP.Hostname, config.Config.HTTP.Port),
			ReadHeaderTimeout: core.ReadHeaderTimeout,
		}

		app.Health.SetReady()

		if err := app.HTTP.StartServer(&srv); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Log.Fatal().Err(err).Msg("shutting down the server")
		}
	}()

	// Setup proxy for existing containers
	app.Log.Info().Msg("Setting up proxy proxies")

	app.ProxyManager.Start()

	// Start watching docker events with cancelable context
	app.ProxyManager.WatchEvents(ctx)

	// Add Routes
	app.Dashboard.AddRoutes()
	core.PprofAddRoutes(app.HTTP)
}

func (app *WebApp) Stop() {
	app.Log.Info().Msg("Shutdown server")

	app.Health.SetNotReady()

	// Shutdown things here
	//
	app.ProxyManager.StopAllProxies()

	app.HTTP.Shutdown()

	app.Log.Info().Msg("Server was shutdown successfully")
}

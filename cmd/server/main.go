package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"simple-license-server/internal/api"
	"simple-license-server/internal/config"
	"simple-license-server/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := storage.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	server := api.NewServerWithOptions(store, logger, cfg.ServerAPIKeys, api.Options{
		RequestTimeout:        cfg.RequestTimeout,
		RateLimitEnabled:      cfg.RateLimitEnabled,
		RateLimitGlobalRPS:    cfg.RateLimitGlobalRPS,
		RateLimitGlobalBurst:  cfg.RateLimitGlobalBurst,
		RateLimitPerIPRPS:     cfg.RateLimitPerIPRPS,
		RateLimitPerIPBurst:   cfg.RateLimitPerIPBurst,
		RateLimitIPTTL:        cfg.RateLimitIPTTL,
		RateLimitMaxIPEntries: cfg.RateLimitMaxIPEntries,
		TrustProxyHeaders:     cfg.TrustProxyHeaders,
	})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("license API server started", "addr", httpServer.Addr)
		errCh <- httpServer.ListenAndServe()
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("http server stopped")
}

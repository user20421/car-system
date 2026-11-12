package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/car-system/car-system/backend-go/internal/auth"
	"github.com/car-system/car-system/backend-go/internal/commands"
	"github.com/car-system/car-system/backend-go/internal/config"
	"github.com/car-system/car-system/backend-go/internal/events"
	"github.com/car-system/car-system/backend-go/internal/httpapi"
	"github.com/car-system/car-system/backend-go/internal/mqttbroker"
	"github.com/car-system/car-system/backend-go/internal/realtime"
	"github.com/car-system/car-system/backend-go/internal/store"
	"github.com/car-system/car-system/backend-go/internal/telemetry"
	"github.com/redis/go-redis/v9"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := store.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		return err
	}
	if err := db.Seed(ctx); err != nil {
		return err
	}
	ca, err := os.ReadFile(cfg.CACertFile)
	if err != nil {
		return err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		return errors.New("invalid CA")
	}
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Username: cfg.Redis.User, Password: cfg.Redis.Password, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: hostOnly(cfg.Redis.Addr)}})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return err
	}
	bus, err := events.Dial(cfg.RabbitMQ.URL, cfg.CACertFile, logger)
	if err != nil {
		return err
	}
	defer bus.Close()
	key, err := config.LoadOrCreateJWTKey(cfg.JWTKeyFile)
	if err != nil {
		return err
	}
	authService := auth.NewService(db, key, "car-system")
	hub := realtime.NewHub()
	tickets := realtime.NewTickets(time.Minute)
	pipeline := telemetry.NewPipeline(db, redisClient, hub, cfg.TelemetryWorkers, cfg.TelemetryQueue, cfg.TelemetryBatch, logger)
	defer pipeline.Close()
	broker, err := mqttbroker.New(cfg, db, pipeline, hub, logger)
	if err != nil {
		return err
	}
	defer broker.Close()
	dispatcher := commands.NewDispatcher(db, broker, hub, logger)
	api := &httpapi.API{Store: db, Auth: authService, Redis: redisClient, Queue: bus, Tickets: tickets, Hub: hub, Pipeline: pipeline, PublicURL: cfg.PublicURL, Log: logger}
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: api.Router(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 120 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, ClientCAs: roots, ClientAuth: tls.VerifyClientCertIfGiven, NextProtos: []string{"h2", "http/1.1"}}}
	errCh := make(chan error, 4)
	go func() {
		logger.Info("MQTTS broker listening", "addr", cfg.MQTTAddr)
		if err := broker.Serve(); err != nil {
			errCh <- err
		}
	}()
	go func() {
		logger.Info("HTTPS server listening", "addr", cfg.HTTPAddr)
		errCh <- httpServer.ListenAndServeTLS(cfg.HTTPCertFile, cfg.HTTPKeyFile)
	}()
	go bus.Relay(ctx, db)
	go dispatcher.ExpireLoop(ctx)
	go func() {
		if err := bus.Consume(ctx, dispatcher); err != nil && ctx.Err() == nil {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			stop()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = httpServer.Shutdown(shutdownCtx)
			return err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}
func hostOnly(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

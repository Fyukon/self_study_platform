package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"learning-roadmap/internal/app"
	"learning-roadmap/internal/config"
	"learning-roadmap/internal/database"
	"learning-roadmap/web"
)

var version = "dev"

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply database migrations and exit")
	seedOnly := flag.Bool("seed-only", false, "apply migrations and seed data, then exit")
	port := flag.String("port", "8080", "localhost port")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load configuration", "error", err)
		os.Exit(1)
	}
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		logger.Error("apply migrations", "error", err)
		os.Exit(1)
	}
	if err := database.EnsureSettings(ctx, db, cfg); err != nil {
		logger.Error("initialize settings", "error", err)
		os.Exit(1)
	}
	if *migrateOnly {
		return
	}
	if err := database.Seed(ctx, db); err != nil {
		logger.Error("import seed", "error", err)
		os.Exit(1)
	}
	if *seedOnly {
		return
	}

	app.Version = version
	server := &http.Server{
		Addr:              net.JoinHostPort("127.0.0.1", *port),
		Handler:           app.New(db, logger, web.Handler()).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	errorsCh := make(chan error, 1)
	go func() {
		logger.Info("server started", "address", server.Addr, "version", version)
		errorsCh <- server.ListenAndServe()
	}()

	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-signals.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown server", "error", err)
		}
	case err := <-errorsCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve http", "error", err)
			os.Exit(1)
		}
	}
}

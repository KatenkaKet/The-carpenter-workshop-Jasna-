package main

import (
	"context"
	"errors"
	"jasna/backend/internal/app"
	"jasna/backend/internal/database"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		slog.Error("server stopped", "error", e)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg, e := app.LoadConfig()
	if e != nil {
		return e
	}
	db, e := database.Open(ctx, cfg.DatabaseURL)
	if e != nil {
		return e
	}
	defer db.Close()
	if e = database.Migrate(ctx, db); e != nil {
		return e
	}
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		slog.Info("migrations applied")
		return nil
	}
	a := app.New(cfg)
	a.DB = db
	if e = a.Bootstrap(ctx); e != nil {
		return e
	}
	server := &http.Server{Addr: cfg.Addr, Handler: a.Router(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	go a.RunJobs(ctx)
	done := make(chan error, 1)
	go func() { slog.Info("listening", "address", cfg.Addr); done <- server.ListenAndServe() }()
	select {
	case e := <-done:
		if errors.Is(e, http.ErrServerClosed) {
			return nil
		}
		return e
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}

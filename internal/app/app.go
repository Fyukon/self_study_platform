package app

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"learning-roadmap/internal/modules/courses"
	"learning-roadmap/internal/modules/roadmap"
	"learning-roadmap/internal/modules/settings"
	"learning-roadmap/internal/shared/httpx"
)

var Version = "dev"

type App struct {
	db       *sql.DB
	logger   *slog.Logger
	mux      *http.ServeMux
	reporter httpx.Reporter
}

func New(db *sql.DB, logger *slog.Logger, frontend http.Handler) *App {
	if logger == nil {
		logger = slog.Default()
	}
	a := &App{
		db:       db,
		logger:   logger,
		mux:      http.NewServeMux(),
		reporter: httpx.NewReporter(logger),
	}
	a.routes(
		settings.New(db, logger),
		roadmap.New(db, logger),
		courses.New(db, logger),
	)
	if frontend != nil {
		a.mux.Handle("/", frontend)
	}
	return a
}

func (a *App) Handler() http.Handler {
	return httpx.Middleware(a.logger, a.mux)
}

func (a *App) routes(settingsHandler *settings.Handler, roadmapHandler *roadmap.Handler, coursesHandler *courses.Handler) {
	a.mux.HandleFunc("GET /api/v1/health", a.health)
	a.mux.HandleFunc("GET /api/v1/version", a.version)
	settingsHandler.RegisterRoutes(a.mux)
	roadmapHandler.RegisterRoutes(a.mux)
	coursesHandler.RegisterRoutes(a.mux)
	a.mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		a.reporter.Problem(w, http.StatusNotFound, "ENDPOINT_NOT_FOUND", "API endpoint was not found", "route_request", 0, nil)
	})
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := a.db.PingContext(ctx); err != nil {
		a.reporter.Problem(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Database is unavailable", "health", 0, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) version(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"version": Version})
}

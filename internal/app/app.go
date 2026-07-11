package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxBodyBytes = 1 << 20

var Version = "dev"

type App struct {
	db     *sql.DB
	logger *slog.Logger
	mux    *http.ServeMux
}

func New(db *sql.DB, logger *slog.Logger, frontend http.Handler) *App {
	if logger == nil {
		logger = slog.Default()
	}
	a := &App{db: db, logger: logger, mux: http.NewServeMux()}
	a.routes()
	if frontend != nil {
		a.mux.Handle("/", frontend)
	}
	return a
}

func (a *App) Handler() http.Handler { return a.middleware(a.mux) }

func (a *App) routes() {
	a.mux.HandleFunc("GET /api/v1/health", a.health)
	a.mux.HandleFunc("GET /api/v1/version", a.version)
	a.mux.HandleFunc("GET /api/v1/settings", a.getSettings)
	a.mux.HandleFunc("PUT /api/v1/settings", a.updateSettings)

	a.mux.HandleFunc("GET /api/v1/directions", a.listDirections)
	a.mux.HandleFunc("POST /api/v1/directions", a.createDirection)
	a.mux.HandleFunc("GET /api/v1/directions/{id}", a.getDirection)
	a.mux.HandleFunc("PUT /api/v1/directions/{id}", a.updateDirection)
	a.mux.HandleFunc("DELETE /api/v1/directions/{id}", a.archiveDirection)
	a.mux.HandleFunc("GET /api/v1/directions/{id}/roadmap", a.roadmapTree)

	a.mux.HandleFunc("POST /api/v1/roadmap/nodes", a.createNode)
	a.mux.HandleFunc("GET /api/v1/roadmap/nodes/{id}", a.getNodeHandler)
	a.mux.HandleFunc("PUT /api/v1/roadmap/nodes/{id}", a.updateNode)
	a.mux.HandleFunc("DELETE /api/v1/roadmap/nodes/{id}", a.deleteNode)
	a.mux.HandleFunc("POST /api/v1/roadmap/nodes/{id}/move", a.moveNode)
	a.mux.HandleFunc("POST /api/v1/roadmap/nodes/{id}/dependencies", a.createDependency)
	a.mux.HandleFunc("DELETE /api/v1/roadmap/nodes/{id}/dependencies/{dependencyId}", a.deleteDependency)
	a.mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		a.problem(w, r, http.StatusNotFound, "ENDPOINT_NOT_FOUND", "API endpoint was not found", "route_request", 0, nil)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (a *App) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := newRequestID()
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		rw := &responseWriter{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				a.logger.Error("http panic", "request_id", requestID, "error", fmt.Sprint(recovered),
					"error_code", "INTERNAL_ERROR", "operation", "http_request")
				if rw.status == 0 {
					writeError(rw, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
				}
			}
			status := rw.status
			if status == 0 {
				status = http.StatusOK
			}
			a.logger.Info("http request", "request_id", requestID, "method", r.Method,
				"path", r.URL.Path, "status", status, "duration_ms", time.Since(started).Milliseconds())
		}()
		next.ServeHTTP(rw, r)
	})
}

func newRequestID() string {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(value[:])
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	writeJSON(w, status, map[string]any{"error": map[string]any{
		"code": code, "message": message, "details": details,
	}})
}

func (a *App) problem(w http.ResponseWriter, r *http.Request, status int, code, message, operation string, entityID int64, err error) {
	args := []any{"error_code", code, "operation", operation}
	if entityID != 0 {
		args = append(args, "entity_id", entityID)
	}
	if err != nil {
		args = append(args, "error", err)
	}
	if status >= 500 {
		a.logger.Error("request failed", args...)
	} else {
		a.logger.Warn("request rejected", args...)
	}
	writeError(w, status, code, message, nil)
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return fmt.Errorf("request body is too large: %w", err)
		}
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func (a *App) decodeOrProblem(w http.ResponseWriter, r *http.Request, dst any, operation string) bool {
	if err := decodeJSON(r, dst); err != nil {
		status := http.StatusBadRequest
		code := "INVALID_JSON"
		message := "Request body must be valid JSON"
		if strings.Contains(err.Error(), "too large") {
			status, code, message = http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "Request body is too large"
		}
		a.problem(w, r, status, code, message, operation, 0, err)
		return false
	}
	return true
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return id, nil
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := a.db.PingContext(ctx); err != nil {
		a.problem(w, r, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Database is unavailable", "health", 0, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": Version})
}

package settings

import (
	"database/sql"
	"log/slog"
	"net/http"

	"learning-roadmap/internal/shared/httpx"
)

type Handler struct {
	db       *sql.DB
	reporter httpx.Reporter
}

func New(db *sql.DB, logger *slog.Logger) *Handler {
	return &Handler{db: db, reporter: httpx.NewReporter(logger)}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/settings", h.getSettings)
	mux.HandleFunc("PUT /api/v1/settings", h.updateSettings)
}

func (h *Handler) problem(w http.ResponseWriter, _ *http.Request, status int, code, message, operation string, entityID int64, err error) {
	h.reporter.Problem(w, status, code, message, operation, entityID, err)
}

func (h *Handler) decodeOrProblem(w http.ResponseWriter, r *http.Request, dst any, operation string) bool {
	return httpx.DecodeOrProblem(h.reporter, w, r, dst, operation)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	httpx.WriteJSON(w, status, value)
}

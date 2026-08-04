package roadmap

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
	mux.HandleFunc("GET /api/v1/directions", h.listDirections)
	mux.HandleFunc("POST /api/v1/directions", h.createDirection)
	mux.HandleFunc("GET /api/v1/directions/{id}", h.getDirection)
	mux.HandleFunc("PUT /api/v1/directions/{id}", h.updateDirection)
	mux.HandleFunc("DELETE /api/v1/directions/{id}", h.archiveDirection)
	mux.HandleFunc("GET /api/v1/directions/{id}/roadmap", h.roadmapTree)

	mux.HandleFunc("POST /api/v1/roadmap/nodes", h.createNode)
	mux.HandleFunc("GET /api/v1/roadmap/nodes/{id}", h.getNodeHandler)
	mux.HandleFunc("PUT /api/v1/roadmap/nodes/{id}", h.updateNode)
	mux.HandleFunc("DELETE /api/v1/roadmap/nodes/{id}", h.deleteNode)
	mux.HandleFunc("POST /api/v1/roadmap/nodes/{id}/move", h.moveNode)
	mux.HandleFunc("POST /api/v1/roadmap/nodes/{id}/dependencies", h.createDependency)
	mux.HandleFunc("DELETE /api/v1/roadmap/nodes/{id}/dependencies/{dependencyId}", h.deleteDependency)
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

func pathID(r *http.Request, name string) (int64, error) {
	return httpx.PathID(r, name)
}

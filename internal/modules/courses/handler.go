package courses

import (
	"database/sql"
	"log/slog"
	"net/http"
	"sync"

	"learning-roadmap/internal/shared/httpx"
)

const (
	CourseModuleNotStarted = courseModuleNotStarted
	CourseModuleInProgress = courseModuleInProgress
)

type ImportPreviewResponse = courseImportPreviewResponse

type Handler struct {
	db             *sql.DB
	reporter       httpx.Reporter
	importMu       sync.Mutex
	importPreviews map[string]courseImportPreview
}

func New(db *sql.DB, logger *slog.Logger) *Handler {
	return &Handler{
		db:             db,
		reporter:       httpx.NewReporter(logger),
		importPreviews: make(map[string]courseImportPreview),
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/courses", h.listCourses)
	mux.HandleFunc("POST /api/v1/courses", h.createCourse)
	mux.HandleFunc("GET /api/v1/courses/{id}", h.getCourse)
	mux.HandleFunc("PUT /api/v1/courses/{id}", h.updateCourse)
	mux.HandleFunc("DELETE /api/v1/courses/{id}", h.archiveCourse)
	mux.HandleFunc("POST /api/v1/courses/{id}/modules", h.createCourseModule)
	mux.HandleFunc("POST /api/v1/courses/import/preview", h.previewCourseImport)
	mux.HandleFunc("POST /api/v1/courses/import", h.importCourse)
	mux.HandleFunc("PUT /api/v1/course-modules/{id}", h.updateCourseModule)
	mux.HandleFunc("DELETE /api/v1/course-modules/{id}", h.deleteCourseModule)
	mux.HandleFunc("POST /api/v1/course-modules/{id}/resources", h.createResource)
	mux.HandleFunc("POST /api/v1/course-modules/{id}/roadmap-links", h.createRoadmapLink)
	mux.HandleFunc("DELETE /api/v1/course-modules/{id}/roadmap-links/{linkId}", h.deleteRoadmapLink)
	mux.HandleFunc("PUT /api/v1/resources/{id}", h.updateResource)
	mux.HandleFunc("DELETE /api/v1/resources/{id}", h.deleteResource)
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

func newRequestID() string {
	return httpx.NewRequestID()
}

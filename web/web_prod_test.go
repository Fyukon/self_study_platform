//go:build production

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesSPAForClientRoute(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/roadmap", nil)
	response := httptest.NewRecorder()

	Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `id="root"`) {
		t.Fatalf("GET /roadmap = %d %q", response.Code, response.Body.String())
	}
}

func TestHandlerServesArchitectureMap(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/architecture.html", nil)
	response := httptest.NewRecorder()

	Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Карта архитектуры") {
		t.Fatalf("GET /architecture.html = %d", response.Code)
	}
}

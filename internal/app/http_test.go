package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"learning-roadmap/internal/config"
	"learning-roadmap/internal/database"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{ApplicationName: "Learning Roadmap", Timezone: "Europe/Moscow", Theme: "system",
		WeekStartsOn: 1, DatabasePath: "test.db", BackupPath: "backups"}
	if err := database.EnsureSettings(context.Background(), db, cfg); err != nil {
		t.Fatal(err)
	}
	return New(db, slog.New(slog.NewTextHandler(io.Discard, nil)), nil).Handler()
}

func request(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var encoded io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		encoded = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, encoded)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func decodeResponse[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response %d %q: %v", response.Code, response.Body.String(), err)
	}
	return value
}

func TestHTTPDirectionAndRoadmapCRUD(t *testing.T) {
	handler := testHandler(t)
	response := request(t, handler, http.MethodPost, "/api/v1/directions", map[string]any{"title": "Go"})
	if response.Code != http.StatusCreated {
		t.Fatalf("create direction: %d %s", response.Code, response.Body.String())
	}
	direction := decodeResponse[Direction](t, response)
	response = request(t, handler, http.MethodPut, "/api/v1/directions/"+strconv.FormatInt(direction.ID, 10), map[string]any{"description": "Backend"})
	if response.Code != http.StatusOK || decodeResponse[Direction](t, response).Description != "Backend" {
		t.Fatalf("update direction: %d %s", response.Code, response.Body.String())
	}

	response = request(t, handler, http.MethodPost, "/api/v1/roadmap/nodes", map[string]any{
		"direction_id": direction.ID, "title": "HTTP", "node_type": "concept",
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("create node: %d %s", response.Code, response.Body.String())
	}
	node := decodeResponse[RoadmapNode](t, response)
	response = request(t, handler, http.MethodPut, "/api/v1/roadmap/nodes/"+strconv.FormatInt(node.ID, 10), map[string]any{
		"status": "practicing", "confidence": 4, "needs_review": true, "next_action": "Write a server",
	})
	if response.Code != http.StatusOK {
		t.Fatalf("update node: %d %s", response.Code, response.Body.String())
	}
	updated := decodeResponse[RoadmapNode](t, response)
	if updated.Status != "practicing" || updated.Confidence != 4 || !updated.NeedsReview {
		t.Fatalf("progress was not persisted: %+v", updated)
	}
	response = request(t, handler, http.MethodGet, "/api/v1/directions/"+strconv.FormatInt(direction.ID, 10)+"/roadmap", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("get tree: %d %s", response.Code, response.Body.String())
	}
	tree := decodeResponse[[]RoadmapNode](t, response)
	if len(tree) != 1 || tree[0].ID != node.ID || tree[0].Children == nil {
		t.Fatalf("unexpected tree: %+v", tree)
	}
	response = request(t, handler, http.MethodDelete, "/api/v1/roadmap/nodes/"+strconv.FormatInt(node.ID, 10), nil)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete node: %d %s", response.Code, response.Body.String())
	}
	response = request(t, handler, http.MethodDelete, "/api/v1/directions/"+strconv.FormatInt(direction.ID, 10), nil)
	if response.Code != http.StatusNoContent {
		t.Fatalf("archive direction: %d %s", response.Code, response.Body.String())
	}
	response = request(t, handler, http.MethodGet, "/api/v1/directions/"+strconv.FormatInt(direction.ID, 10), nil)
	if response.Code != http.StatusOK || !decodeResponse[Direction](t, response).IsArchived {
		t.Fatalf("direction was not archived: %d %s", response.Code, response.Body.String())
	}
}

func TestHTTPErrorFormat(t *testing.T) {
	handler := testHandler(t)
	response := request(t, handler, http.MethodGet, "/api/v1/roadmap/nodes/999", nil)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	body = decodeResponse[struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}](t, response)
	if body.Error.Code != "ROADMAP_NODE_NOT_FOUND" || body.Error.Message == "" || body.Error.Details == nil {
		t.Fatalf("unexpected error: %+v", body.Error)
	}
}

func TestHTTPMoveAndDependencies(t *testing.T) {
	handler := testHandler(t)
	directionResponse := request(t, handler, http.MethodPost, "/api/v1/directions", map[string]any{"title": "Go"})
	direction := decodeResponse[Direction](t, directionResponse)
	createNode := func(title string, parentID *int64) RoadmapNode {
		body := map[string]any{"direction_id": direction.ID, "title": title}
		if parentID != nil {
			body["parent_id"] = *parentID
		}
		response := request(t, handler, http.MethodPost, "/api/v1/roadmap/nodes", body)
		if response.Code != http.StatusCreated {
			t.Fatalf("create %s: %d %s", title, response.Code, response.Body.String())
		}
		return decodeResponse[RoadmapNode](t, response)
	}
	root := createNode("Root", nil)
	other := createNode("Other", nil)
	child := createNode("Child", &root.ID)

	response := request(t, handler, http.MethodPost, "/api/v1/roadmap/nodes/"+strconv.FormatInt(root.ID, 10)+"/move", map[string]any{"parent_id": child.ID})
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("cycle accepted: %d %s", response.Code, response.Body.String())
	}
	response = request(t, handler, http.MethodPost, "/api/v1/roadmap/nodes/"+strconv.FormatInt(child.ID, 10)+"/move", map[string]any{"parent_id": other.ID, "position": 2})
	if response.Code != http.StatusOK {
		t.Fatalf("move child: %d %s", response.Code, response.Body.String())
	}
	moved := decodeResponse[RoadmapNode](t, response)
	if moved.ParentID == nil || *moved.ParentID != other.ID || moved.Position != 2 {
		t.Fatalf("unexpected move result: %+v", moved)
	}
	dependencyPath := "/api/v1/roadmap/nodes/" + strconv.FormatInt(child.ID, 10) + "/dependencies"
	response = request(t, handler, http.MethodPost, dependencyPath, map[string]any{"depends_on_node_id": root.ID})
	if response.Code != http.StatusCreated {
		t.Fatalf("create dependency: %d %s", response.Code, response.Body.String())
	}
	dependency := decodeResponse[NodeDependency](t, response)
	response = request(t, handler, http.MethodPost, dependencyPath, map[string]any{"depends_on_node_id": root.ID})
	if response.Code != http.StatusConflict {
		t.Fatalf("duplicate dependency: %d %s", response.Code, response.Body.String())
	}
	response = request(t, handler, http.MethodDelete, dependencyPath+"/"+strconv.FormatInt(dependency.ID, 10), nil)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete dependency: %d %s", response.Code, response.Body.String())
	}
}

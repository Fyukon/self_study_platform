package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func courseErrorCode(t *testing.T, responseBody []byte) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(responseBody, &body); err != nil {
		t.Fatalf("decode error response %q: %v", string(responseBody), err)
	}
	return body.Error.Code
}

func TestHTTPCoursesAndMaterialsCRUD(t *testing.T) {
	handler := testHandler(t)

	directionResponse := request(t, handler, http.MethodPost, "/api/v1/directions", map[string]any{"title": "Go"})
	if directionResponse.Code != http.StatusCreated {
		t.Fatalf("create direction: %d %s", directionResponse.Code, directionResponse.Body.String())
	}
	direction := decodeResponse[Direction](t, directionResponse)
	nodeResponse := request(t, handler, http.MethodPost, "/api/v1/roadmap/nodes", map[string]any{
		"direction_id": direction.ID,
		"title":        "HTTP-сервер",
	})
	if nodeResponse.Code != http.StatusCreated {
		t.Fatalf("create roadmap node: %d %s", nodeResponse.Code, nodeResponse.Body.String())
	}
	node := decodeResponse[RoadmapNode](t, nodeResponse)

	courseResponse := request(t, handler, http.MethodPost, "/api/v1/courses", map[string]any{
		"title":       "Практический Go",
		"description": "Курс для backend-разработки",
		"provider":    "Self study",
		"source_url":  "https://example.com/course",
	})
	if courseResponse.Code != http.StatusCreated {
		t.Fatalf("create course: %d %s", courseResponse.Code, courseResponse.Body.String())
	}
	course := decodeResponse[Course](t, courseResponse)
	if course.Title != "Практический Go" || len(course.Modules) != 0 {
		t.Fatalf("unexpected created course: %+v", course)
	}

	modulePath := "/api/v1/courses/" + strconv.FormatInt(course.ID, 10) + "/modules"
	moduleResponse := request(t, handler, http.MethodPost, modulePath, map[string]any{
		"title":       "HTTP и сети",
		"description": "Разбираем серверы и протоколы",
	})
	if moduleResponse.Code != http.StatusCreated {
		t.Fatalf("create course module: %d %s", moduleResponse.Code, moduleResponse.Body.String())
	}
	module := decodeResponse[CourseModule](t, moduleResponse)
	if module.Status != courseModuleNotStarted {
		t.Fatalf("default module status = %q", module.Status)
	}

	resourcePath := "/api/v1/course-modules/" + strconv.FormatInt(module.ID, 10) + "/resources"
	bookResponse := request(t, handler, http.MethodPost, resourcePath, map[string]any{
		"title":         "The Go Programming Language",
		"resource_type": "book",
		"url":           "https://example.com/go-book",
		"note":          "Главы 1–3",
	})
	if bookResponse.Code != http.StatusCreated {
		t.Fatalf("create book resource: %d %s", bookResponse.Code, bookResponse.Body.String())
	}
	book := decodeResponse[CourseResource](t, bookResponse)
	localResponse := request(t, handler, http.MethodPost, resourcePath, map[string]any{
		"title":         "Локальные заметки",
		"resource_type": "local_file",
		"local_path":    "/home/user/notes/go-http.md",
	})
	if localResponse.Code != http.StatusCreated {
		t.Fatalf("create local resource: %d %s", localResponse.Code, localResponse.Body.String())
	}
	local := decodeResponse[CourseResource](t, localResponse)
	if book.URL == "" || local.LocalPath == "" || local.URL != "" {
		t.Fatalf("resource locations were not persisted: book=%+v local=%+v", book, local)
	}

	linkPath := "/api/v1/course-modules/" + strconv.FormatInt(module.ID, 10) + "/roadmap-links"
	linkResponse := request(t, handler, http.MethodPost, linkPath, map[string]any{"node_id": node.ID})
	if linkResponse.Code != http.StatusCreated {
		t.Fatalf("create roadmap link: %d %s", linkResponse.Code, linkResponse.Body.String())
	}
	link := decodeResponse[CourseModuleRoadmapLink](t, linkResponse)
	if link.NodeTitle != node.Title || link.DirectionTitle != direction.Title {
		t.Fatalf("unexpected roadmap link: %+v", link)
	}
	secondModuleResponse := request(t, handler, http.MethodPost, modulePath, map[string]any{
		"title": "Linux-практика",
	})
	if secondModuleResponse.Code != http.StatusCreated {
		t.Fatalf("create second course module: %d %s", secondModuleResponse.Code, secondModuleResponse.Body.String())
	}
	secondModule := decodeResponse[CourseModule](t, secondModuleResponse)
	secondResourceResponse := request(t, handler, http.MethodPost,
		"/api/v1/course-modules/"+strconv.FormatInt(secondModule.ID, 10)+"/resources", map[string]any{
			"title":         "Linux manual",
			"resource_type": "article",
			"url":           "https://example.com/linux-manual",
		})
	if secondResourceResponse.Code != http.StatusCreated {
		t.Fatalf("create second module resource: %d %s", secondResourceResponse.Code, secondResourceResponse.Body.String())
	}
	secondLinkResponse := request(t, handler, http.MethodPost,
		"/api/v1/course-modules/"+strconv.FormatInt(secondModule.ID, 10)+"/roadmap-links", map[string]any{"node_id": node.ID})
	if secondLinkResponse.Code != http.StatusCreated {
		t.Fatalf("create second module roadmap link: %d %s", secondLinkResponse.Code, secondLinkResponse.Body.String())
	}
	duplicateLinkResponse := request(t, handler, http.MethodPost, linkPath, map[string]any{"node_id": node.ID})
	if duplicateLinkResponse.Code != http.StatusConflict || courseErrorCode(t, duplicateLinkResponse.Body.Bytes()) != "ROADMAP_LINK_EXISTS" {
		t.Fatalf("duplicate roadmap link: %d %s", duplicateLinkResponse.Code, duplicateLinkResponse.Body.String())
	}

	moduleUpdateResponse := request(t, handler, http.MethodPut, "/api/v1/course-modules/"+strconv.FormatInt(module.ID, 10), map[string]any{
		"status": "in_progress",
	})
	if moduleUpdateResponse.Code != http.StatusOK || decodeResponse[CourseModule](t, moduleUpdateResponse).Status != courseModuleInProgress {
		t.Fatalf("update module: %d %s", moduleUpdateResponse.Code, moduleUpdateResponse.Body.String())
	}
	resourceUpdateResponse := request(t, handler, http.MethodPut, "/api/v1/resources/"+strconv.FormatInt(book.ID, 10), map[string]any{
		"note": "Прочитать до следующей сессии",
	})
	if resourceUpdateResponse.Code != http.StatusOK || decodeResponse[CourseResource](t, resourceUpdateResponse).Note == "" {
		t.Fatalf("update resource: %d %s", resourceUpdateResponse.Code, resourceUpdateResponse.Body.String())
	}

	getResponse := request(t, handler, http.MethodGet, "/api/v1/courses/"+strconv.FormatInt(course.ID, 10), nil)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get course: %d %s", getResponse.Code, getResponse.Body.String())
	}
	loaded := decodeResponse[Course](t, getResponse)
	if len(loaded.Modules) != 2 || len(loaded.Modules[0].Resources) != 2 || len(loaded.Modules[0].RoadmapLinks) != 1 ||
		len(loaded.Modules[1].Resources) != 1 || len(loaded.Modules[1].RoadmapLinks) != 1 {
		t.Fatalf("nested course data was not loaded: %+v", loaded)
	}
	listResponse := request(t, handler, http.MethodGet, "/api/v1/courses", nil)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list courses: %d %s", listResponse.Code, listResponse.Body.String())
	}
	list := decodeResponse[[]CourseSummary](t, listResponse)
	if len(list) != 1 || list[0].ModuleCount != 2 || list[0].ResourceCount != 3 {
		t.Fatalf("unexpected course summary: %+v", list)
	}

	if response := request(t, handler, http.MethodDelete, "/api/v1/course-modules/"+strconv.FormatInt(module.ID, 10)+"/roadmap-links/"+strconv.FormatInt(link.ID, 10), nil); response.Code != http.StatusNoContent {
		t.Fatalf("delete roadmap link: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, handler, http.MethodDelete, "/api/v1/resources/"+strconv.FormatInt(local.ID, 10), nil); response.Code != http.StatusNoContent {
		t.Fatalf("delete local resource: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, handler, http.MethodDelete, "/api/v1/courses/"+strconv.FormatInt(course.ID, 10), nil); response.Code != http.StatusNoContent {
		t.Fatalf("archive course: %d %s", response.Code, response.Body.String())
	}
	if response := request(t, handler, http.MethodPost, modulePath, map[string]any{"title": "Не добавится"}); response.Code != http.StatusNotFound {
		t.Fatalf("archived course accepted a new module: %d %s", response.Code, response.Body.String())
	}
}

func TestHTTPCourseImportRequiresPreviewAndSupportsMarkdownAndJSON(t *testing.T) {
	handler := testHandler(t)

	withoutPreview := request(t, handler, http.MethodPost, "/api/v1/courses/import", map[string]any{})
	if withoutPreview.Code != http.StatusUnprocessableEntity || courseErrorCode(t, withoutPreview.Body.Bytes()) != "PREVIEW_REQUIRED" {
		t.Fatalf("import without preview: %d %s", withoutPreview.Code, withoutPreview.Body.String())
	}

	jsonContent := `{"title":"Go с нуля","description":"Путь по основам","provider":"Book club","source_url":"https://example.com/go","modules":[{"title":"Основы","description":"Типы и функции","resources":[{"title":"Книга","resource_type":"book","url":"https://example.com/book","note":"Главы 1–2"},{"title":"Репозиторий","resource_type":"repository","url":"https://github.com/example/go"}]}]}`
	previewResponse := request(t, handler, http.MethodPost, "/api/v1/courses/import/preview", map[string]any{
		"format":  "json",
		"content": jsonContent,
	})
	if previewResponse.Code != http.StatusOK {
		t.Fatalf("preview JSON: %d %s", previewResponse.Code, previewResponse.Body.String())
	}
	preview := decodeResponse[courseImportPreviewResponse](t, previewResponse)
	if preview.PreviewID == "" || preview.Summary.Modules != 1 || preview.Summary.Resources != 2 {
		t.Fatalf("unexpected JSON preview: %+v", preview)
	}
	importResponse := request(t, handler, http.MethodPost, "/api/v1/courses/import", map[string]any{"preview_id": preview.PreviewID})
	if importResponse.Code != http.StatusCreated {
		t.Fatalf("import JSON: %d %s", importResponse.Code, importResponse.Body.String())
	}
	imported := decodeResponse[Course](t, importResponse)
	if imported.Title != "Go с нуля" || len(imported.Modules) != 1 || len(imported.Modules[0].Resources) != 2 {
		t.Fatalf("unexpected imported JSON course: %+v", imported)
	}
	reusedPreview := request(t, handler, http.MethodPost, "/api/v1/courses/import", map[string]any{"preview_id": preview.PreviewID})
	if reusedPreview.Code != http.StatusConflict || courseErrorCode(t, reusedPreview.Body.Bytes()) != "PREVIEW_EXPIRED" {
		t.Fatalf("reused preview: %d %s", reusedPreview.Code, reusedPreview.Body.String())
	}

	markdown := "# Linux для разработчика\n\nПрактический список материалов.\n\n## Файлы и процессы\n\nЧитаем и закрепляем.\n\n- [book] The Linux Command Line — https://example.com/linux-book\n- [local] Мои заметки — /home/user/linux.md\n"
	markdownPreviewResponse := request(t, handler, http.MethodPost, "/api/v1/courses/import/preview", map[string]any{
		"format":  "markdown",
		"content": markdown,
	})
	if markdownPreviewResponse.Code != http.StatusOK {
		t.Fatalf("preview Markdown: %d %s", markdownPreviewResponse.Code, markdownPreviewResponse.Body.String())
	}
	markdownPreview := decodeResponse[courseImportPreviewResponse](t, markdownPreviewResponse)
	if markdownPreview.Course.Title != "Linux для разработчика" || markdownPreview.Course.Description == "" || markdownPreview.Summary.Resources != 2 {
		t.Fatalf("unexpected Markdown preview: %+v", markdownPreview)
	}
	markdownImportResponse := request(t, handler, http.MethodPost, "/api/v1/courses/import", map[string]any{"preview_id": markdownPreview.PreviewID})
	if markdownImportResponse.Code != http.StatusCreated {
		t.Fatalf("import Markdown: %d %s", markdownImportResponse.Code, markdownImportResponse.Body.String())
	}
	markdownImported := decodeResponse[Course](t, markdownImportResponse)
	if markdownImported.Modules[0].Resources[1].ResourceType != "local_file" {
		t.Fatalf("Markdown local resource was not normalized: %+v", markdownImported.Modules[0].Resources[1])
	}

	invalidPreview := request(t, handler, http.MethodPost, "/api/v1/courses/import/preview", map[string]any{
		"format":  "json",
		"content": `{"title":"Без модулей","modules":[]}`,
	})
	if invalidPreview.Code != http.StatusUnprocessableEntity || courseErrorCode(t, invalidPreview.Body.Bytes()) != "INVALID_COURSE_IMPORT" {
		t.Fatalf("invalid preview: %d %s", invalidPreview.Code, invalidPreview.Body.String())
	}
}

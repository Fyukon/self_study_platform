package courses

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	courseModuleNotStarted = "not_started"
	courseModuleInProgress = "in_progress"
	courseModuleCompleted  = "completed"
	courseImportTTL        = 15 * time.Minute
)

var validCourseModuleStatuses = map[string]bool{
	courseModuleNotStarted: true,
	courseModuleInProgress: true,
	courseModuleCompleted:  true,
}

var validResourceTypes = map[string]bool{
	"book":       true,
	"video":      true,
	"repository": true,
	"article":    true,
	"link":       true,
	"local_file": true,
}

type CourseSummary struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Provider      string `json:"provider"`
	SourceURL     string `json:"source_url"`
	Position      int    `json:"position"`
	IsArchived    bool   `json:"is_archived"`
	ModuleCount   int    `json:"module_count"`
	ResourceCount int    `json:"resource_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type Course struct {
	ID          int64          `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Provider    string         `json:"provider"`
	SourceURL   string         `json:"source_url"`
	Position    int            `json:"position"`
	IsArchived  bool           `json:"is_archived"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
	Modules     []CourseModule `json:"modules"`
}

type CourseModule struct {
	ID           int64                     `json:"id"`
	CourseID     int64                     `json:"course_id"`
	Title        string                    `json:"title"`
	Description  string                    `json:"description"`
	Status       string                    `json:"status"`
	Position     int                       `json:"position"`
	CreatedAt    string                    `json:"created_at"`
	UpdatedAt    string                    `json:"updated_at"`
	Resources    []CourseResource          `json:"resources"`
	RoadmapLinks []CourseModuleRoadmapLink `json:"roadmap_links"`
}

type CourseResource struct {
	ID           int64  `json:"id"`
	ModuleID     int64  `json:"module_id"`
	Title        string `json:"title"`
	ResourceType string `json:"resource_type"`
	URL          string `json:"url"`
	LocalPath    string `json:"local_path"`
	Note         string `json:"note"`
	Position     int    `json:"position"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type CourseModuleRoadmapLink struct {
	ID             int64  `json:"id"`
	ModuleID       int64  `json:"module_id"`
	NodeID         int64  `json:"node_id"`
	NodeTitle      string `json:"node_title"`
	DirectionID    int64  `json:"direction_id"`
	DirectionTitle string `json:"direction_title"`
	CreatedAt      string `json:"created_at"`
}

const courseColumns = `id, title, description, provider, source_url, position,
	is_archived, created_at, updated_at`

func (a *Handler) listCourses(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), `SELECT c.id, c.title, c.description, c.provider,
		c.source_url, c.position, c.is_archived, c.created_at, c.updated_at,
		(SELECT COUNT(*) FROM course_modules m WHERE m.course_id = c.id),
		(SELECT COUNT(*) FROM resources resource
			JOIN course_modules m ON m.id = resource.module_id WHERE m.course_id = c.id)
		FROM courses c ORDER BY c.is_archived, c.position, c.id`)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSES_READ_FAILED", "Could not read courses", "list_courses", 0, err)
		return
	}
	defer rows.Close()
	courses := []CourseSummary{}
	for rows.Next() {
		var course CourseSummary
		if err := rows.Scan(&course.ID, &course.Title, &course.Description, &course.Provider,
			&course.SourceURL, &course.Position, &course.IsArchived, &course.CreatedAt,
			&course.UpdatedAt, &course.ModuleCount, &course.ResourceCount); err != nil {
			a.problem(w, r, http.StatusInternalServerError, "COURSES_READ_FAILED", "Could not read courses", "list_courses", 0, err)
			return
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSES_READ_FAILED", "Could not read courses", "list_courses", 0, err)
		return
	}
	writeJSON(w, http.StatusOK, courses)
}

func (a *Handler) courseByID(ctx context.Context, id int64) (Course, error) {
	var course Course
	err := a.db.QueryRowContext(ctx, `SELECT `+courseColumns+` FROM courses WHERE id = ?`, id).Scan(
		&course.ID, &course.Title, &course.Description, &course.Provider, &course.SourceURL,
		&course.Position, &course.IsArchived, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		return Course{}, err
	}
	course.Modules = []CourseModule{}
	modules, err := a.db.QueryContext(ctx, `SELECT id, course_id, title, description, status,
		position, created_at, updated_at FROM course_modules
		WHERE course_id = ? ORDER BY position, id`, id)
	if err != nil {
		return Course{}, err
	}
	moduleByID := make(map[int64]*CourseModule)
	for modules.Next() {
		var module CourseModule
		if err := modules.Scan(&module.ID, &module.CourseID, &module.Title, &module.Description,
			&module.Status, &module.Position, &module.CreatedAt, &module.UpdatedAt); err != nil {
			modules.Close()
			return Course{}, err
		}
		module.Resources = []CourseResource{}
		module.RoadmapLinks = []CourseModuleRoadmapLink{}
		course.Modules = append(course.Modules, module)
		moduleByID[module.ID] = &course.Modules[len(course.Modules)-1]
	}
	if err := modules.Close(); err != nil {
		return Course{}, err
	}
	if err := modules.Err(); err != nil {
		return Course{}, err
	}

	resources, err := a.db.QueryContext(ctx, `SELECT resource.id, resource.module_id, resource.title,
		resource.resource_type, resource.url, resource.local_path, resource.note, resource.position,
		resource.created_at, resource.updated_at
		FROM resources resource JOIN course_modules m ON m.id = resource.module_id
		WHERE m.course_id = ? ORDER BY m.position, m.id, resource.position, resource.id`, id)
	if err != nil {
		return Course{}, err
	}
	for resources.Next() {
		var resource CourseResource
		if err := resources.Scan(&resource.ID, &resource.ModuleID, &resource.Title,
			&resource.ResourceType, &resource.URL, &resource.LocalPath, &resource.Note,
			&resource.Position, &resource.CreatedAt, &resource.UpdatedAt); err != nil {
			resources.Close()
			return Course{}, err
		}
		if module := moduleByID[resource.ModuleID]; module != nil {
			module.Resources = append(module.Resources, resource)
		}
	}
	if err := resources.Close(); err != nil {
		return Course{}, err
	}
	if err := resources.Err(); err != nil {
		return Course{}, err
	}

	links, err := a.db.QueryContext(ctx, `SELECT link.id, link.module_id, link.node_id,
		n.title, d.id, d.title, link.created_at
		FROM course_module_roadmap_links link
		JOIN course_modules m ON m.id = link.module_id
		JOIN roadmap_nodes n ON n.id = link.node_id
		JOIN directions d ON d.id = n.direction_id
		WHERE m.course_id = ? ORDER BY m.position, m.id, link.id`, id)
	if err != nil {
		return Course{}, err
	}
	for links.Next() {
		var link CourseModuleRoadmapLink
		if err := links.Scan(&link.ID, &link.ModuleID, &link.NodeID, &link.NodeTitle,
			&link.DirectionID, &link.DirectionTitle, &link.CreatedAt); err != nil {
			links.Close()
			return Course{}, err
		}
		if module := moduleByID[link.ModuleID]; module != nil {
			module.RoadmapLinks = append(module.RoadmapLinks, link)
		}
	}
	if err := links.Close(); err != nil {
		return Course{}, err
	}
	if err := links.Err(); err != nil {
		return Course{}, err
	}
	return course, nil
}

func (a *Handler) getCourse(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_COURSE_ID", "Course id is invalid", "get_course", 0, err)
		return
	}
	course, err := a.courseByID(r.Context(), id)
	if err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "COURSE_NOT_FOUND", "Course was not found", "get_course", id, err)
		return
	}
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_READ_FAILED", "Could not read course", "get_course", id, err)
		return
	}
	writeJSON(w, http.StatusOK, course)
}

type courseCreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Provider    string `json:"provider"`
	SourceURL   string `json:"source_url"`
	Position    *int   `json:"position"`
}

func validateCourse(title, sourceURL string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is required")
	}
	if err := validateHTTPURL(sourceURL, "source_url"); err != nil {
		return err
	}
	return nil
}

func validateHTTPURL(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be an http(s) URL", field)
	}
	return nil
}

func (a *Handler) createCourse(w http.ResponseWriter, r *http.Request) {
	var input courseCreateInput
	if !a.decodeOrProblem(w, r, &input, "create_course") {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Provider = strings.TrimSpace(input.Provider)
	input.SourceURL = strings.TrimSpace(input.SourceURL)
	if err := validateCourse(input.Title, input.SourceURL); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_COURSE", err.Error(), "create_course", 0, err)
		return
	}
	position := 0
	if input.Position != nil {
		position = *input.Position
	} else if err := a.db.QueryRowContext(r.Context(), `SELECT COALESCE(MAX(position) + 1, 0) FROM courses`).Scan(&position); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_CREATE_FAILED", "Could not create course", "create_course", 0, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `INSERT INTO courses(title, description, provider, source_url, position)
		VALUES (?, ?, ?, ?, ?)`, input.Title, input.Description, input.Provider, input.SourceURL, position)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_CREATE_FAILED", "Could not create course", "create_course", 0, err)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_CREATE_FAILED", "Could not create course", "create_course", 0, err)
		return
	}
	course, err := a.courseByID(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_READ_FAILED", "Could not read created course", "create_course", id, err)
		return
	}
	writeJSON(w, http.StatusCreated, course)
}

type courseUpdateInput struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Provider    *string `json:"provider"`
	SourceURL   *string `json:"source_url"`
	Position    *int    `json:"position"`
	IsArchived  *bool   `json:"is_archived"`
}

func (a *Handler) updateCourse(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_COURSE_ID", "Course id is invalid", "update_course", 0, err)
		return
	}
	var input courseUpdateInput
	if !a.decodeOrProblem(w, r, &input, "update_course") {
		return
	}
	course, err := a.courseByID(r.Context(), id)
	if err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "COURSE_NOT_FOUND", "Course was not found", "update_course", id, err)
		return
	}
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_READ_FAILED", "Could not read course", "update_course", id, err)
		return
	}
	if input.Title != nil {
		course.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		course.Description = strings.TrimSpace(*input.Description)
	}
	if input.Provider != nil {
		course.Provider = strings.TrimSpace(*input.Provider)
	}
	if input.SourceURL != nil {
		course.SourceURL = strings.TrimSpace(*input.SourceURL)
	}
	if input.Position != nil {
		course.Position = *input.Position
	}
	if input.IsArchived != nil {
		course.IsArchived = *input.IsArchived
	}
	if err := validateCourse(course.Title, course.SourceURL); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_COURSE", err.Error(), "update_course", id, err)
		return
	}
	_, err = a.db.ExecContext(r.Context(), `UPDATE courses SET title = ?, description = ?, provider = ?,
		source_url = ?, position = ?, is_archived = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?`, course.Title, course.Description, course.Provider, course.SourceURL,
		course.Position, course.IsArchived, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_UPDATE_FAILED", "Could not update course", "update_course", id, err)
		return
	}
	course, err = a.courseByID(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_READ_FAILED", "Could not read updated course", "update_course", id, err)
		return
	}
	writeJSON(w, http.StatusOK, course)
}

func (a *Handler) archiveCourse(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_COURSE_ID", "Course id is invalid", "archive_course", 0, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `UPDATE courses SET is_archived = 1,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_ARCHIVE_FAILED", "Could not archive course", "archive_course", id, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		a.problem(w, r, http.StatusNotFound, "COURSE_NOT_FOUND", "Course was not found", "archive_course", id, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type courseModuleCreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Position    *int   `json:"position"`
}

func validateCourseModule(title, status string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is required")
	}
	if !validCourseModuleStatuses[status] {
		return fmt.Errorf("invalid status")
	}
	return nil
}

func (a *Handler) readModule(ctx context.Context, id int64) (CourseModule, error) {
	var module CourseModule
	err := a.db.QueryRowContext(ctx, `SELECT id, course_id, title, description, status,
		position, created_at, updated_at FROM course_modules WHERE id = ?`, id).Scan(
		&module.ID, &module.CourseID, &module.Title, &module.Description, &module.Status,
		&module.Position, &module.CreatedAt, &module.UpdatedAt)
	if err != nil {
		return CourseModule{}, err
	}
	module.Resources = []CourseResource{}
	module.RoadmapLinks = []CourseModuleRoadmapLink{}
	return module, nil
}

func (a *Handler) createCourseModule(w http.ResponseWriter, r *http.Request) {
	courseID, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_COURSE_ID", "Course id is invalid", "create_course_module", 0, err)
		return
	}
	var input courseModuleCreateInput
	if !a.decodeOrProblem(w, r, &input, "create_course_module") {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Status == "" {
		input.Status = courseModuleNotStarted
	}
	if err := validateCourseModule(input.Title, input.Status); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_COURSE_MODULE", err.Error(), "create_course_module", 0, err)
		return
	}
	var exists bool
	if err := a.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM courses WHERE id = ? AND is_archived = 0)`, courseID).Scan(&exists); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_READ_FAILED", "Could not read course", "create_course_module", courseID, err)
		return
	}
	if !exists {
		a.problem(w, r, http.StatusNotFound, "COURSE_NOT_FOUND", "Course was not found", "create_course_module", courseID, nil)
		return
	}
	position := 0
	if input.Position != nil {
		position = *input.Position
	} else if err := a.db.QueryRowContext(r.Context(), `SELECT COALESCE(MAX(position) + 1, 0)
		FROM course_modules WHERE course_id = ?`, courseID).Scan(&position); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_CREATE_FAILED", "Could not create course module", "create_course_module", courseID, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `INSERT INTO course_modules(course_id, title, description, status, position)
		VALUES (?, ?, ?, ?, ?)`, courseID, input.Title, input.Description, input.Status, position)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_CREATE_FAILED", "Could not create course module", "create_course_module", courseID, err)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_CREATE_FAILED", "Could not create course module", "create_course_module", courseID, err)
		return
	}
	module, err := a.readModule(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_READ_FAILED", "Could not read created course module", "create_course_module", id, err)
		return
	}
	writeJSON(w, http.StatusCreated, module)
}

type courseModuleUpdateInput struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Position    *int    `json:"position"`
}

func (a *Handler) updateCourseModule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_COURSE_MODULE_ID", "Course module id is invalid", "update_course_module", 0, err)
		return
	}
	var input courseModuleUpdateInput
	if !a.decodeOrProblem(w, r, &input, "update_course_module") {
		return
	}
	module, err := a.readModule(r.Context(), id)
	if err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "COURSE_MODULE_NOT_FOUND", "Course module was not found", "update_course_module", id, err)
		return
	}
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_READ_FAILED", "Could not read course module", "update_course_module", id, err)
		return
	}
	if input.Title != nil {
		module.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		module.Description = strings.TrimSpace(*input.Description)
	}
	if input.Status != nil {
		module.Status = *input.Status
	}
	if input.Position != nil {
		module.Position = *input.Position
	}
	if err := validateCourseModule(module.Title, module.Status); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_COURSE_MODULE", err.Error(), "update_course_module", id, err)
		return
	}
	_, err = a.db.ExecContext(r.Context(), `UPDATE course_modules SET title = ?, description = ?, status = ?,
		position = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`,
		module.Title, module.Description, module.Status, module.Position, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_UPDATE_FAILED", "Could not update course module", "update_course_module", id, err)
		return
	}
	module, err = a.readModule(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_READ_FAILED", "Could not read updated course module", "update_course_module", id, err)
		return
	}
	writeJSON(w, http.StatusOK, module)
}

func (a *Handler) deleteCourseModule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_COURSE_MODULE_ID", "Course module id is invalid", "delete_course_module", 0, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `DELETE FROM course_modules WHERE id = ?`, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_DELETE_FAILED", "Could not delete course module", "delete_course_module", id, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		a.problem(w, r, http.StatusNotFound, "COURSE_MODULE_NOT_FOUND", "Course module was not found", "delete_course_module", id, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type resourceCreateInput struct {
	Title        string `json:"title"`
	ResourceType string `json:"resource_type"`
	URL          string `json:"url"`
	LocalPath    string `json:"local_path"`
	Note         string `json:"note"`
	Position     *int   `json:"position"`
}

func normalizeResourceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "repo":
		return "repository"
	case "local", "file":
		return "local_file"
	case "url":
		return "link"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func validateResource(title, resourceType, resourceURL, localPath string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is required")
	}
	if !validResourceTypes[resourceType] {
		return fmt.Errorf("invalid resource_type")
	}
	if resourceType == "local_file" {
		if strings.TrimSpace(localPath) == "" || strings.TrimSpace(resourceURL) != "" {
			return fmt.Errorf("local_file requires local_path and no url")
		}
		return nil
	}
	if strings.TrimSpace(localPath) != "" {
		return fmt.Errorf("non-local resources cannot have local_path")
	}
	if err := validateHTTPURL(resourceURL, "url"); err != nil {
		return err
	}
	if strings.TrimSpace(resourceURL) == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

func (a *Handler) readResource(ctx context.Context, id int64) (CourseResource, error) {
	var resource CourseResource
	err := a.db.QueryRowContext(ctx, `SELECT id, module_id, title, resource_type, url, local_path,
		note, position, created_at, updated_at FROM resources WHERE id = ?`, id).Scan(
		&resource.ID, &resource.ModuleID, &resource.Title, &resource.ResourceType, &resource.URL,
		&resource.LocalPath, &resource.Note, &resource.Position, &resource.CreatedAt, &resource.UpdatedAt)
	return resource, err
}

func (a *Handler) createResource(w http.ResponseWriter, r *http.Request) {
	moduleID, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_COURSE_MODULE_ID", "Course module id is invalid", "create_resource", 0, err)
		return
	}
	var input resourceCreateInput
	if !a.decodeOrProblem(w, r, &input, "create_resource") {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.ResourceType = normalizeResourceType(input.ResourceType)
	input.URL = strings.TrimSpace(input.URL)
	input.LocalPath = strings.TrimSpace(input.LocalPath)
	input.Note = strings.TrimSpace(input.Note)
	if input.ResourceType == "" {
		input.ResourceType = "link"
	}
	if err := validateResource(input.Title, input.ResourceType, input.URL, input.LocalPath); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_RESOURCE", err.Error(), "create_resource", moduleID, err)
		return
	}
	var exists bool
	if err := a.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM course_modules WHERE id = ?)`, moduleID).Scan(&exists); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_READ_FAILED", "Could not read course module", "create_resource", moduleID, err)
		return
	}
	if !exists {
		a.problem(w, r, http.StatusNotFound, "COURSE_MODULE_NOT_FOUND", "Course module was not found", "create_resource", moduleID, nil)
		return
	}
	position := 0
	if input.Position != nil {
		position = *input.Position
	} else if err := a.db.QueryRowContext(r.Context(), `SELECT COALESCE(MAX(position) + 1, 0)
		FROM resources WHERE module_id = ?`, moduleID).Scan(&position); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "RESOURCE_CREATE_FAILED", "Could not create resource", "create_resource", moduleID, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `INSERT INTO resources(module_id, title, resource_type, url,
		local_path, note, position) VALUES (?, ?, ?, ?, ?, ?, ?)`, moduleID, input.Title,
		input.ResourceType, input.URL, input.LocalPath, input.Note, position)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "RESOURCE_CREATE_FAILED", "Could not create resource", "create_resource", moduleID, err)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "RESOURCE_CREATE_FAILED", "Could not create resource", "create_resource", moduleID, err)
		return
	}
	resource, err := a.readResource(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "RESOURCE_READ_FAILED", "Could not read created resource", "create_resource", id, err)
		return
	}
	writeJSON(w, http.StatusCreated, resource)
}

type resourceUpdateInput struct {
	Title        *string `json:"title"`
	ResourceType *string `json:"resource_type"`
	URL          *string `json:"url"`
	LocalPath    *string `json:"local_path"`
	Note         *string `json:"note"`
	Position     *int    `json:"position"`
}

func (a *Handler) updateResource(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_RESOURCE_ID", "Resource id is invalid", "update_resource", 0, err)
		return
	}
	var input resourceUpdateInput
	if !a.decodeOrProblem(w, r, &input, "update_resource") {
		return
	}
	resource, err := a.readResource(r.Context(), id)
	if err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "Resource was not found", "update_resource", id, err)
		return
	}
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "RESOURCE_READ_FAILED", "Could not read resource", "update_resource", id, err)
		return
	}
	if input.Title != nil {
		resource.Title = strings.TrimSpace(*input.Title)
	}
	if input.ResourceType != nil {
		resource.ResourceType = normalizeResourceType(*input.ResourceType)
	}
	if input.URL != nil {
		resource.URL = strings.TrimSpace(*input.URL)
	}
	if input.LocalPath != nil {
		resource.LocalPath = strings.TrimSpace(*input.LocalPath)
	}
	if input.Note != nil {
		resource.Note = strings.TrimSpace(*input.Note)
	}
	if input.Position != nil {
		resource.Position = *input.Position
	}
	if err := validateResource(resource.Title, resource.ResourceType, resource.URL, resource.LocalPath); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_RESOURCE", err.Error(), "update_resource", id, err)
		return
	}
	_, err = a.db.ExecContext(r.Context(), `UPDATE resources SET title = ?, resource_type = ?, url = ?,
		local_path = ?, note = ?, position = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?`, resource.Title, resource.ResourceType, resource.URL, resource.LocalPath,
		resource.Note, resource.Position, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "RESOURCE_UPDATE_FAILED", "Could not update resource", "update_resource", id, err)
		return
	}
	resource, err = a.readResource(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "RESOURCE_READ_FAILED", "Could not read updated resource", "update_resource", id, err)
		return
	}
	writeJSON(w, http.StatusOK, resource)
}

func (a *Handler) deleteResource(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_RESOURCE_ID", "Resource id is invalid", "delete_resource", 0, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `DELETE FROM resources WHERE id = ?`, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "RESOURCE_DELETE_FAILED", "Could not delete resource", "delete_resource", id, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		a.problem(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "Resource was not found", "delete_resource", id, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type roadmapLinkCreateInput struct {
	NodeID int64 `json:"node_id"`
}

func (a *Handler) readRoadmapLink(ctx context.Context, id int64) (CourseModuleRoadmapLink, error) {
	var link CourseModuleRoadmapLink
	err := a.db.QueryRowContext(ctx, `SELECT link.id, link.module_id, link.node_id, n.title,
		d.id, d.title, link.created_at FROM course_module_roadmap_links link
		JOIN roadmap_nodes n ON n.id = link.node_id JOIN directions d ON d.id = n.direction_id
		WHERE link.id = ?`, id).Scan(&link.ID, &link.ModuleID, &link.NodeID, &link.NodeTitle,
		&link.DirectionID, &link.DirectionTitle, &link.CreatedAt)
	return link, err
}

func (a *Handler) createRoadmapLink(w http.ResponseWriter, r *http.Request) {
	moduleID, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_COURSE_MODULE_ID", "Course module id is invalid", "create_roadmap_link", 0, err)
		return
	}
	var input roadmapLinkCreateInput
	if !a.decodeOrProblem(w, r, &input, "create_roadmap_link") {
		return
	}
	if input.NodeID <= 0 {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_ROADMAP_NODE_ID", "node_id is required", "create_roadmap_link", moduleID, nil)
		return
	}
	var moduleExists, nodeExists bool
	if err := a.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM course_modules WHERE id = ?)`, moduleID).Scan(&moduleExists); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_MODULE_READ_FAILED", "Could not read course module", "create_roadmap_link", moduleID, err)
		return
	}
	if !moduleExists {
		a.problem(w, r, http.StatusNotFound, "COURSE_MODULE_NOT_FOUND", "Course module was not found", "create_roadmap_link", moduleID, nil)
		return
	}
	if err := a.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM roadmap_nodes WHERE id = ?)`, input.NodeID).Scan(&nodeExists); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read roadmap node", "create_roadmap_link", input.NodeID, err)
		return
	}
	if !nodeExists {
		a.problem(w, r, http.StatusNotFound, "ROADMAP_NODE_NOT_FOUND", "Roadmap node was not found", "create_roadmap_link", input.NodeID, nil)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `INSERT INTO course_module_roadmap_links(module_id, node_id)
		VALUES (?, ?) ON CONFLICT(module_id, node_id) DO NOTHING`, moduleID, input.NodeID)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_LINK_CREATE_FAILED", "Could not link roadmap node", "create_roadmap_link", moduleID, err)
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		a.problem(w, r, http.StatusConflict, "ROADMAP_LINK_EXISTS", "Roadmap node is already linked", "create_roadmap_link", moduleID, nil)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_LINK_CREATE_FAILED", "Could not create roadmap link", "create_roadmap_link", moduleID, err)
		return
	}
	link, err := a.readRoadmapLink(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_LINK_READ_FAILED", "Could not read roadmap link", "create_roadmap_link", id, err)
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

func (a *Handler) deleteRoadmapLink(w http.ResponseWriter, r *http.Request) {
	moduleID, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_COURSE_MODULE_ID", "Course module id is invalid", "delete_roadmap_link", 0, err)
		return
	}
	linkID, err := pathID(r, "linkId")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_ROADMAP_LINK_ID", "Roadmap link id is invalid", "delete_roadmap_link", moduleID, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `DELETE FROM course_module_roadmap_links
		WHERE id = ? AND module_id = ?`, linkID, moduleID)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_LINK_DELETE_FAILED", "Could not delete roadmap link", "delete_roadmap_link", linkID, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		a.problem(w, r, http.StatusNotFound, "ROADMAP_LINK_NOT_FOUND", "Roadmap link was not found", "delete_roadmap_link", linkID, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type courseImportDocument struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Provider    string               `json:"provider"`
	SourceURL   string               `json:"source_url"`
	Modules     []courseImportModule `json:"modules"`
}

type courseImportModule struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Resources   []courseImportResource `json:"resources"`
}

type courseImportResource struct {
	Title        string `json:"title"`
	ResourceType string `json:"resource_type"`
	URL          string `json:"url"`
	LocalPath    string `json:"local_path"`
	Note         string `json:"note"`
}

type courseImportPreview struct {
	Format    string
	document  courseImportDocument
	expiresAt time.Time
}

type courseImportRequest struct {
	Format  string `json:"format"`
	Content string `json:"content"`
}

type courseImportCommitRequest struct {
	PreviewID string `json:"preview_id"`
}

type courseImportPreviewResponse struct {
	PreviewID string               `json:"preview_id"`
	Format    string               `json:"format"`
	Course    courseImportDocument `json:"course"`
	Summary   courseImportSummary  `json:"summary"`
	ExpiresAt string               `json:"expires_at"`
}

type courseImportSummary struct {
	Modules   int `json:"modules"`
	Resources int `json:"resources"`
}

func (a *Handler) storeCoursePreview(format string, document courseImportDocument) string {
	now := time.Now()
	previewID := newRequestID()
	a.importMu.Lock()
	for id, preview := range a.importPreviews {
		if !preview.expiresAt.After(now) {
			delete(a.importPreviews, id)
		}
	}
	a.importPreviews[previewID] = courseImportPreview{
		Format:    format,
		document:  document,
		expiresAt: now.Add(courseImportTTL),
	}
	a.importMu.Unlock()
	return previewID
}

func (a *Handler) takeCoursePreview(previewID string) (courseImportPreview, bool) {
	a.importMu.Lock()
	preview, ok := a.importPreviews[previewID]
	if ok {
		delete(a.importPreviews, previewID)
	}
	a.importMu.Unlock()
	if !ok || !preview.expiresAt.After(time.Now()) {
		return courseImportPreview{}, false
	}
	return preview, true
}

func (a *Handler) previewCourseImport(w http.ResponseWriter, r *http.Request) {
	var input courseImportRequest
	if !a.decodeOrProblem(w, r, &input, "preview_course_import") {
		return
	}
	format := strings.ToLower(strings.TrimSpace(input.Format))
	document, err := parseCourseImport(format, input.Content)
	if err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_COURSE_IMPORT", err.Error(), "preview_course_import", 0, err)
		return
	}
	previewID := a.storeCoursePreview(format, document)
	resources := 0
	for _, module := range document.Modules {
		resources += len(module.Resources)
	}
	writeJSON(w, http.StatusOK, courseImportPreviewResponse{
		PreviewID: previewID,
		Format:    format,
		Course:    document,
		Summary:   courseImportSummary{Modules: len(document.Modules), Resources: resources},
		ExpiresAt: time.Now().Add(courseImportTTL).UTC().Format(time.RFC3339),
	})
}

func parseCourseImport(format, content string) (courseImportDocument, error) {
	if strings.TrimSpace(content) == "" {
		return courseImportDocument{}, fmt.Errorf("content is required")
	}
	switch format {
	case "json":
		var document courseImportDocument
		decoder := json.NewDecoder(strings.NewReader(content))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&document); err != nil {
			return courseImportDocument{}, fmt.Errorf("invalid JSON: %w", err)
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			if err == nil {
				return courseImportDocument{}, fmt.Errorf("content must contain one JSON object")
			}
			return courseImportDocument{}, fmt.Errorf("invalid JSON: %w", err)
		}
		return normalizeAndValidateImport(&document)
	case "markdown", "md":
		document, err := parseMarkdownCourse(content)
		if err != nil {
			return courseImportDocument{}, err
		}
		return normalizeAndValidateImport(&document)
	default:
		return courseImportDocument{}, fmt.Errorf("format must be markdown or json")
	}
}

func parseMarkdownCourse(content string) (courseImportDocument, error) {
	content = strings.TrimPrefix(strings.TrimSpace(content), "\uFEFF")
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	document := courseImportDocument{}
	var courseDescription []string
	var moduleDescription []string
	var current *courseImportModule
	flushModule := func() {
		if current == nil {
			return
		}
		current.Description = cleanImportText(moduleDescription)
		document.Modules = append(document.Modules, *current)
		current = nil
		moduleDescription = nil
	}
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		switch {
		case strings.HasPrefix(line, "# "):
			if document.Title != "" {
				return courseImportDocument{}, fmt.Errorf("markdown must contain one top-level # title")
			}
			document.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		case strings.HasPrefix(line, "## "):
			flushModule()
			current = &courseImportModule{Title: strings.TrimSpace(strings.TrimPrefix(line, "## "))}
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			if current == nil {
				return courseImportDocument{}, fmt.Errorf("resource bullet must be inside a ## module")
			}
			resource, err := parseMarkdownResource(strings.TrimSpace(line[2:]))
			if err != nil {
				return courseImportDocument{}, err
			}
			current.Resources = append(current.Resources, resource)
		default:
			if line == "" {
				continue
			}
			if current == nil {
				courseDescription = append(courseDescription, line)
			} else {
				moduleDescription = append(moduleDescription, line)
			}
		}
	}
	flushModule()
	document.Description = cleanImportText(courseDescription)
	return document, nil
}

func parseMarkdownResource(value string) (courseImportResource, error) {
	resource := courseImportResource{ResourceType: "link"}
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") {
		closing := strings.Index(value, "]")
		if closing < 0 {
			return courseImportResource{}, fmt.Errorf("resource type marker is not closed")
		}
		resource.ResourceType = normalizeResourceType(value[1:closing])
		value = strings.TrimSpace(value[closing+1:])
	}
	separators := []string{" — ", " | ", " - "}
	separator := ""
	for _, candidate := range separators {
		if strings.Contains(value, candidate) {
			separator = candidate
			break
		}
	}
	if separator == "" {
		return courseImportResource{}, fmt.Errorf("resource must use «Название — URL или путь»")
	}
	parts := strings.SplitN(value, separator, 2)
	resource.Title = strings.TrimSpace(parts[0])
	location := strings.TrimSpace(parts[1])
	if resource.ResourceType == "local_file" {
		resource.LocalPath = location
	} else {
		resource.URL = location
	}
	return resource, nil
}

func cleanImportText(lines []string) string {
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func normalizeAndValidateImport(document *courseImportDocument) (courseImportDocument, error) {
	document.Title = strings.TrimSpace(document.Title)
	document.Description = strings.TrimSpace(document.Description)
	document.Provider = strings.TrimSpace(document.Provider)
	document.SourceURL = strings.TrimSpace(document.SourceURL)
	if err := validateCourse(document.Title, document.SourceURL); err != nil {
		return courseImportDocument{}, err
	}
	if len(document.Modules) == 0 {
		return courseImportDocument{}, fmt.Errorf("at least one module is required")
	}
	for moduleIndex := range document.Modules {
		module := &document.Modules[moduleIndex]
		module.Title = strings.TrimSpace(module.Title)
		module.Description = strings.TrimSpace(module.Description)
		if err := validateCourseModule(module.Title, courseModuleNotStarted); err != nil {
			return courseImportDocument{}, fmt.Errorf("module %d: %w", moduleIndex+1, err)
		}
		for resourceIndex := range module.Resources {
			resource := &module.Resources[resourceIndex]
			resource.Title = strings.TrimSpace(resource.Title)
			resource.ResourceType = normalizeResourceType(resource.ResourceType)
			resource.URL = strings.TrimSpace(resource.URL)
			resource.LocalPath = strings.TrimSpace(resource.LocalPath)
			resource.Note = strings.TrimSpace(resource.Note)
			if resource.ResourceType == "" {
				resource.ResourceType = "link"
			}
			if err := validateResource(resource.Title, resource.ResourceType, resource.URL, resource.LocalPath); err != nil {
				return courseImportDocument{}, fmt.Errorf("module %d resource %d: %w", moduleIndex+1, resourceIndex+1, err)
			}
		}
	}
	return *document, nil
}

func (a *Handler) importCourse(w http.ResponseWriter, r *http.Request) {
	var input courseImportCommitRequest
	if !a.decodeOrProblem(w, r, &input, "import_course") {
		return
	}
	input.PreviewID = strings.TrimSpace(input.PreviewID)
	if input.PreviewID == "" {
		a.problem(w, r, http.StatusUnprocessableEntity, "PREVIEW_REQUIRED", "A valid preview_id is required before import", "import_course", 0, nil)
		return
	}
	preview, ok := a.takeCoursePreview(input.PreviewID)
	if !ok {
		a.problem(w, r, http.StatusConflict, "PREVIEW_EXPIRED", "Import preview is missing or expired", "import_course", 0, nil)
		return
	}
	course, err := a.insertImportedCourse(r.Context(), preview.document)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "COURSE_IMPORT_FAILED", "Could not import course", "import_course", 0, err)
		return
	}
	writeJSON(w, http.StatusCreated, course)
}

func (a *Handler) insertImportedCourse(ctx context.Context, document courseImportDocument) (Course, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Course{}, err
	}
	defer tx.Rollback()
	var position int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position) + 1, 0) FROM courses`).Scan(&position); err != nil {
		return Course{}, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO courses(title, description, provider, source_url, position)
		VALUES (?, ?, ?, ?, ?)`, document.Title, document.Description, document.Provider, document.SourceURL, position)
	if err != nil {
		return Course{}, err
	}
	courseID, err := result.LastInsertId()
	if err != nil {
		return Course{}, err
	}
	for moduleIndex, module := range document.Modules {
		result, err := tx.ExecContext(ctx, `INSERT INTO course_modules(course_id, title, description, status, position)
			VALUES (?, ?, ?, ?, ?)`, courseID, module.Title, module.Description, courseModuleNotStarted, moduleIndex)
		if err != nil {
			return Course{}, err
		}
		moduleID, err := result.LastInsertId()
		if err != nil {
			return Course{}, err
		}
		for resourceIndex, resource := range module.Resources {
			if _, err := tx.ExecContext(ctx, `INSERT INTO resources(module_id, title, resource_type, url,
				local_path, note, position) VALUES (?, ?, ?, ?, ?, ?, ?)`, moduleID, resource.Title,
				resource.ResourceType, resource.URL, resource.LocalPath, resource.Note, resourceIndex); err != nil {
				return Course{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return Course{}, err
	}
	return a.courseByID(ctx, courseID)
}

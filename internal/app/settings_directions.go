package app

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

const settingsColumns = `id, application_name, timezone, week_starts_on, database_path,
	backup_path, theme, created_at, updated_at`

func (a *App) readSettings(r *http.Request) (Settings, error) {
	var settings Settings
	err := a.db.QueryRowContext(r.Context(), `SELECT `+settingsColumns+` FROM settings WHERE id = 1`).Scan(
		&settings.ID, &settings.ApplicationName, &settings.Timezone, &settings.WeekStartsOn,
		&settings.DatabasePath, &settings.BackupPath, &settings.Theme,
		&settings.CreatedAt, &settings.UpdatedAt)
	return settings, err
}

func (a *App) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := a.readSettings(r)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "SETTINGS_READ_FAILED", "Could not read settings", "get_settings", 1, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

type settingsUpdate struct {
	ApplicationName *string `json:"application_name"`
	Timezone        *string `json:"timezone"`
	WeekStartsOn    *int    `json:"week_starts_on"`
	DatabasePath    *string `json:"database_path"`
	BackupPath      *string `json:"backup_path"`
	Theme           *string `json:"theme"`
}

func (a *App) updateSettings(w http.ResponseWriter, r *http.Request) {
	var input settingsUpdate
	if !a.decodeOrProblem(w, r, &input, "update_settings") {
		return
	}
	settings, err := a.readSettings(r)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "SETTINGS_READ_FAILED", "Could not read settings", "update_settings", 1, err)
		return
	}
	if input.ApplicationName != nil {
		settings.ApplicationName = strings.TrimSpace(*input.ApplicationName)
	}
	if input.Timezone != nil {
		settings.Timezone = *input.Timezone
	}
	if input.WeekStartsOn != nil {
		settings.WeekStartsOn = *input.WeekStartsOn
	}
	if input.DatabasePath != nil {
		settings.DatabasePath = *input.DatabasePath
	}
	if input.BackupPath != nil {
		settings.BackupPath = *input.BackupPath
	}
	if input.Theme != nil {
		settings.Theme = *input.Theme
	}
	if settings.ApplicationName == "" || settings.DatabasePath == "" || settings.BackupPath == "" {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_SETTINGS", "Application name and paths are required", "update_settings", 1, nil)
		return
	}
	if _, err := time.LoadLocation(settings.Timezone); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_TIMEZONE", "Timezone is invalid", "update_settings", 1, err)
		return
	}
	if settings.WeekStartsOn < 1 || settings.WeekStartsOn > 7 {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_WEEK_START", "week_starts_on must be between 1 and 7", "update_settings", 1, nil)
		return
	}
	if settings.Theme != "light" && settings.Theme != "dark" && settings.Theme != "system" {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_THEME", "Theme must be light, dark, or system", "update_settings", 1, nil)
		return
	}
	_, err = a.db.ExecContext(r.Context(), `UPDATE settings SET application_name = ?, timezone = ?,
		week_starts_on = ?, database_path = ?, backup_path = ?, theme = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = 1`,
		settings.ApplicationName, settings.Timezone, settings.WeekStartsOn, settings.DatabasePath,
		settings.BackupPath, settings.Theme)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "SETTINGS_UPDATE_FAILED", "Could not update settings", "update_settings", 1, err)
		return
	}
	settings, err = a.readSettings(r)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "SETTINGS_READ_FAILED", "Could not read settings", "update_settings", 1, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

const directionColumns = `id, title, description, icon, position, is_archived, created_at, updated_at`

func scanDirection(row scanner) (Direction, error) {
	var direction Direction
	err := row.Scan(&direction.ID, &direction.Title, &direction.Description, &direction.Icon,
		&direction.Position, &direction.IsArchived, &direction.CreatedAt, &direction.UpdatedAt)
	return direction, err
}

func (a *App) listDirections(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), `SELECT `+directionColumns+` FROM directions
		ORDER BY is_archived, position, id`)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTIONS_READ_FAILED", "Could not read directions", "list_directions", 0, err)
		return
	}
	defer rows.Close()
	directions := []Direction{}
	for rows.Next() {
		direction, err := scanDirection(rows)
		if err != nil {
			a.problem(w, r, http.StatusInternalServerError, "DIRECTIONS_READ_FAILED", "Could not read directions", "list_directions", 0, err)
			return
		}
		directions = append(directions, direction)
	}
	if err := rows.Err(); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTIONS_READ_FAILED", "Could not read directions", "list_directions", 0, err)
		return
	}
	writeJSON(w, http.StatusOK, directions)
}

type directionCreate struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Position    *int   `json:"position"`
}

func (a *App) createDirection(w http.ResponseWriter, r *http.Request) {
	var input directionCreate
	if !a.decodeOrProblem(w, r, &input, "create_direction") {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_DIRECTION", "Title is required", "create_direction", 0, nil)
		return
	}
	position := 0
	if input.Position != nil {
		position = *input.Position
	} else if err := a.db.QueryRowContext(r.Context(), `SELECT COALESCE(MAX(position) + 1, 0) FROM directions`).Scan(&position); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_CREATE_FAILED", "Could not create direction", "create_direction", 0, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `INSERT INTO directions(title, description, icon, position) VALUES (?, ?, ?, ?)`,
		input.Title, input.Description, input.Icon, position)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_CREATE_FAILED", "Could not create direction", "create_direction", 0, err)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_CREATE_FAILED", "Could not create direction", "create_direction", 0, err)
		return
	}
	direction, err := a.directionByID(r, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "create_direction", id, err)
		return
	}
	writeJSON(w, http.StatusCreated, direction)
}

func (a *App) directionByID(r *http.Request, id int64) (Direction, error) {
	return scanDirection(a.db.QueryRowContext(r.Context(), `SELECT `+directionColumns+` FROM directions WHERE id = ?`, id))
}

func (a *App) getDirection(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_DIRECTION_ID", "Direction id is invalid", "get_direction", 0, err)
		return
	}
	direction, err := a.directionByID(r, id)
	if err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "DIRECTION_NOT_FOUND", "Direction was not found", "get_direction", id, err)
		return
	}
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "get_direction", id, err)
		return
	}
	writeJSON(w, http.StatusOK, direction)
}

type directionUpdate struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	Position    *int    `json:"position"`
	IsArchived  *bool   `json:"is_archived"`
}

func (a *App) updateDirection(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_DIRECTION_ID", "Direction id is invalid", "update_direction", 0, err)
		return
	}
	var input directionUpdate
	if !a.decodeOrProblem(w, r, &input, "update_direction") {
		return
	}
	direction, err := a.directionByID(r, id)
	if err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "DIRECTION_NOT_FOUND", "Direction was not found", "update_direction", id, err)
		return
	}
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "update_direction", id, err)
		return
	}
	if input.Title != nil {
		direction.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		direction.Description = *input.Description
	}
	if input.Icon != nil {
		direction.Icon = *input.Icon
	}
	if input.Position != nil {
		direction.Position = *input.Position
	}
	if input.IsArchived != nil {
		direction.IsArchived = *input.IsArchived
	}
	if direction.Title == "" {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_DIRECTION", "Title is required", "update_direction", id, nil)
		return
	}
	_, err = a.db.ExecContext(r.Context(), `UPDATE directions SET title = ?, description = ?, icon = ?,
		position = ?, is_archived = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`,
		direction.Title, direction.Description, direction.Icon, direction.Position, direction.IsArchived, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_UPDATE_FAILED", "Could not update direction", "update_direction", id, err)
		return
	}
	direction, err = a.directionByID(r, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "update_direction", id, err)
		return
	}
	writeJSON(w, http.StatusOK, direction)
}

func (a *App) archiveDirection(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_DIRECTION_ID", "Direction id is invalid", "archive_direction", 0, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `UPDATE directions SET is_archived = 1,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_ARCHIVE_FAILED", "Could not archive direction", "archive_direction", id, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		a.problem(w, r, http.StatusNotFound, "DIRECTION_NOT_FOUND", "Direction was not found", "archive_direction", id, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

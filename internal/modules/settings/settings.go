package settings

import (
	"net/http"
	"strings"
	"time"
)

const settingsColumns = `id, application_name, timezone, week_starts_on, database_path,
	backup_path, theme, created_at, updated_at`

func (h *Handler) readSettings(r *http.Request) (Settings, error) {
	var settings Settings
	err := h.db.QueryRowContext(r.Context(), `SELECT `+settingsColumns+` FROM settings WHERE id = 1`).Scan(
		&settings.ID, &settings.ApplicationName, &settings.Timezone, &settings.WeekStartsOn,
		&settings.DatabasePath, &settings.BackupPath, &settings.Theme,
		&settings.CreatedAt, &settings.UpdatedAt)
	return settings, err
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.readSettings(r)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "SETTINGS_READ_FAILED", "Could not read settings", "get_settings", 1, err)
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

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var input settingsUpdate
	if !h.decodeOrProblem(w, r, &input, "update_settings") {
		return
	}
	settings, err := h.readSettings(r)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "SETTINGS_READ_FAILED", "Could not read settings", "update_settings", 1, err)
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
		h.problem(w, r, http.StatusUnprocessableEntity, "INVALID_SETTINGS", "Application name and paths are required", "update_settings", 1, nil)
		return
	}
	if _, err := time.LoadLocation(settings.Timezone); err != nil {
		h.problem(w, r, http.StatusUnprocessableEntity, "INVALID_TIMEZONE", "Timezone is invalid", "update_settings", 1, err)
		return
	}
	if settings.WeekStartsOn < 1 || settings.WeekStartsOn > 7 {
		h.problem(w, r, http.StatusUnprocessableEntity, "INVALID_WEEK_START", "week_starts_on must be between 1 and 7", "update_settings", 1, nil)
		return
	}
	if settings.Theme != "light" && settings.Theme != "dark" && settings.Theme != "system" {
		h.problem(w, r, http.StatusUnprocessableEntity, "INVALID_THEME", "Theme must be light, dark, or system", "update_settings", 1, nil)
		return
	}
	_, err = h.db.ExecContext(r.Context(), `UPDATE settings SET application_name = ?, timezone = ?,
		week_starts_on = ?, database_path = ?, backup_path = ?, theme = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = 1`,
		settings.ApplicationName, settings.Timezone, settings.WeekStartsOn, settings.DatabasePath,
		settings.BackupPath, settings.Theme)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "SETTINGS_UPDATE_FAILED", "Could not update settings", "update_settings", 1, err)
		return
	}
	settings, err = h.readSettings(r)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "SETTINGS_READ_FAILED", "Could not read settings", "update_settings", 1, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

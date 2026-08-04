package roadmap

import (
	"database/sql"
	"net/http"
	"strings"
)

const directionColumns = `id, title, description, icon, position, is_archived, created_at, updated_at`

func scanDirection(row scanner) (Direction, error) {
	var direction Direction
	err := row.Scan(&direction.ID, &direction.Title, &direction.Description, &direction.Icon,
		&direction.Position, &direction.IsArchived, &direction.CreatedAt, &direction.UpdatedAt)
	return direction, err
}

func (h *Handler) listDirections(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `SELECT `+directionColumns+` FROM directions
		ORDER BY is_archived, position, id`)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTIONS_READ_FAILED", "Could not read directions", "list_directions", 0, err)
		return
	}
	defer rows.Close()
	directions := []Direction{}
	for rows.Next() {
		direction, err := scanDirection(rows)
		if err != nil {
			h.problem(w, r, http.StatusInternalServerError, "DIRECTIONS_READ_FAILED", "Could not read directions", "list_directions", 0, err)
			return
		}
		directions = append(directions, direction)
	}
	if err := rows.Err(); err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTIONS_READ_FAILED", "Could not read directions", "list_directions", 0, err)
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

func (h *Handler) createDirection(w http.ResponseWriter, r *http.Request) {
	var input directionCreate
	if !h.decodeOrProblem(w, r, &input, "create_direction") {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		h.problem(w, r, http.StatusUnprocessableEntity, "INVALID_DIRECTION", "Title is required", "create_direction", 0, nil)
		return
	}
	position := 0
	if input.Position != nil {
		position = *input.Position
	} else if err := h.db.QueryRowContext(r.Context(), `SELECT COALESCE(MAX(position) + 1, 0) FROM directions`).Scan(&position); err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTION_CREATE_FAILED", "Could not create direction", "create_direction", 0, err)
		return
	}
	result, err := h.db.ExecContext(r.Context(), `INSERT INTO directions(title, description, icon, position) VALUES (?, ?, ?, ?)`,
		input.Title, input.Description, input.Icon, position)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTION_CREATE_FAILED", "Could not create direction", "create_direction", 0, err)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTION_CREATE_FAILED", "Could not create direction", "create_direction", 0, err)
		return
	}
	direction, err := h.directionByID(r, id)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "create_direction", id, err)
		return
	}
	writeJSON(w, http.StatusCreated, direction)
}

func (h *Handler) directionByID(r *http.Request, id int64) (Direction, error) {
	return scanDirection(h.db.QueryRowContext(r.Context(), `SELECT `+directionColumns+` FROM directions WHERE id = ?`, id))
}

func (h *Handler) getDirection(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		h.problem(w, r, http.StatusBadRequest, "INVALID_DIRECTION_ID", "Direction id is invalid", "get_direction", 0, err)
		return
	}
	direction, err := h.directionByID(r, id)
	if err == sql.ErrNoRows {
		h.problem(w, r, http.StatusNotFound, "DIRECTION_NOT_FOUND", "Direction was not found", "get_direction", id, err)
		return
	}
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "get_direction", id, err)
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

func (h *Handler) updateDirection(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		h.problem(w, r, http.StatusBadRequest, "INVALID_DIRECTION_ID", "Direction id is invalid", "update_direction", 0, err)
		return
	}
	var input directionUpdate
	if !h.decodeOrProblem(w, r, &input, "update_direction") {
		return
	}
	direction, err := h.directionByID(r, id)
	if err == sql.ErrNoRows {
		h.problem(w, r, http.StatusNotFound, "DIRECTION_NOT_FOUND", "Direction was not found", "update_direction", id, err)
		return
	}
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "update_direction", id, err)
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
		h.problem(w, r, http.StatusUnprocessableEntity, "INVALID_DIRECTION", "Title is required", "update_direction", id, nil)
		return
	}
	_, err = h.db.ExecContext(r.Context(), `UPDATE directions SET title = ?, description = ?, icon = ?,
		position = ?, is_archived = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`,
		direction.Title, direction.Description, direction.Icon, direction.Position, direction.IsArchived, id)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTION_UPDATE_FAILED", "Could not update direction", "update_direction", id, err)
		return
	}
	direction, err = h.directionByID(r, id)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "update_direction", id, err)
		return
	}
	writeJSON(w, http.StatusOK, direction)
}

func (h *Handler) archiveDirection(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		h.problem(w, r, http.StatusBadRequest, "INVALID_DIRECTION_ID", "Direction id is invalid", "archive_direction", 0, err)
		return
	}
	result, err := h.db.ExecContext(r.Context(), `UPDATE directions SET is_archived = 1,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`, id)
	if err != nil {
		h.problem(w, r, http.StatusInternalServerError, "DIRECTION_ARCHIVE_FAILED", "Could not archive direction", "archive_direction", id, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		h.problem(w, r, http.StatusNotFound, "DIRECTION_NOT_FOUND", "Direction was not found", "archive_direction", id, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

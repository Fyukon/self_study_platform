package roadmap

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (a *Handler) nodeByID(ctx context.Context, id int64) (RoadmapNode, error) {
	node, err := scanNode(a.db.QueryRowContext(ctx, `SELECT `+nodeColumns+` FROM roadmap_nodes WHERE id = ?`, id))
	if err != nil {
		return RoadmapNode{}, err
	}
	dependencies, err := a.dependencies(ctx, id)
	if err != nil {
		return RoadmapNode{}, err
	}
	node.Dependencies = dependencies
	return node, nil
}

func (a *Handler) dependencies(ctx context.Context, nodeID int64) ([]NodeDependency, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, node_id, depends_on_node_id, created_at
		FROM node_dependencies WHERE node_id = ? ORDER BY id`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	dependencies := []NodeDependency{}
	for rows.Next() {
		var dependency NodeDependency
		if err := rows.Scan(&dependency.ID, &dependency.NodeID, &dependency.DependsOnNodeID, &dependency.CreatedAt); err != nil {
			return nil, err
		}
		dependencies = append(dependencies, dependency)
	}
	return dependencies, rows.Err()
}

func (a *Handler) roadmapTree(w http.ResponseWriter, r *http.Request) {
	directionID, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_DIRECTION_ID", "Direction id is invalid", "roadmap_tree", 0, err)
		return
	}
	if _, err := a.directionByID(r, directionID); err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "DIRECTION_NOT_FOUND", "Direction was not found", "roadmap_tree", directionID, err)
		return
	} else if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "roadmap_tree", directionID, err)
		return
	}
	rows, err := a.db.QueryContext(r.Context(), `SELECT `+nodeColumns+` FROM roadmap_nodes
		WHERE direction_id = ? ORDER BY position, id`, directionID)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_READ_FAILED", "Could not read roadmap", "roadmap_tree", directionID, err)
		return
	}
	defer rows.Close()
	byID := make(map[int64]*RoadmapNode)
	ordered := []*RoadmapNode{}
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			a.problem(w, r, http.StatusInternalServerError, "ROADMAP_READ_FAILED", "Could not read roadmap", "roadmap_tree", directionID, err)
			return
		}
		copy := node
		byID[node.ID] = &copy
		ordered = append(ordered, &copy)
	}
	if err := rows.Err(); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_READ_FAILED", "Could not read roadmap", "roadmap_tree", directionID, err)
		return
	}
	dependencyRows, err := a.db.QueryContext(r.Context(), `SELECT d.id, d.node_id, d.depends_on_node_id, d.created_at
		FROM node_dependencies d JOIN roadmap_nodes n ON n.id = d.node_id
		WHERE n.direction_id = ? ORDER BY d.id`, directionID)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_READ_FAILED", "Could not read roadmap", "roadmap_tree", directionID, err)
		return
	}
	for dependencyRows.Next() {
		var dependency NodeDependency
		if err := dependencyRows.Scan(&dependency.ID, &dependency.NodeID, &dependency.DependsOnNodeID, &dependency.CreatedAt); err != nil {
			dependencyRows.Close()
			a.problem(w, r, http.StatusInternalServerError, "ROADMAP_READ_FAILED", "Could not read roadmap", "roadmap_tree", directionID, err)
			return
		}
		if node := byID[dependency.NodeID]; node != nil {
			node.Dependencies = append(node.Dependencies, dependency)
		}
	}
	if err := dependencyRows.Close(); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_READ_FAILED", "Could not read roadmap", "roadmap_tree", directionID, err)
		return
	}
	roots := []*RoadmapNode{}
	for _, node := range ordered {
		if node.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		if parent := byID[*node.ParentID]; parent != nil {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	writeJSON(w, http.StatusOK, roots)
}

type nodeCreate struct {
	DirectionID    int64   `json:"direction_id"`
	ParentID       *int64  `json:"parent_id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	NodeType       string  `json:"node_type"`
	Status         string  `json:"status"`
	NeedsReview    bool    `json:"needs_review"`
	Confidence     int     `json:"confidence"`
	Position       *int    `json:"position"`
	NextAction     string  `json:"next_action"`
	TargetDate     *string `json:"target_date"`
	LastReviewedAt *string `json:"last_reviewed_at"`
	EstimatedHours float64 `json:"estimated_hours"`
}

func (a *Handler) createNode(w http.ResponseWriter, r *http.Request) {
	var input nodeCreate
	if !a.decodeOrProblem(w, r, &input, "create_node") {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	if input.NodeType == "" {
		input.NodeType = "concept"
	}
	if input.Status == "" {
		input.Status = "not_started"
	}
	if input.Confidence == 0 {
		input.Confidence = 1
	}
	if err := validateNode(input.Title, input.NodeType, input.Status, input.Confidence, input.EstimatedHours); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_ROADMAP_NODE", err.Error(), "create_node", 0, err)
		return
	}
	if input.DirectionID <= 0 {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_DIRECTION_ID", "direction_id is required", "create_node", 0, nil)
		return
	}
	var directionExists bool
	if err := a.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM directions WHERE id = ?)`, input.DirectionID).Scan(&directionExists); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DIRECTION_READ_FAILED", "Could not read direction", "create_node", input.DirectionID, err)
		return
	}
	if !directionExists {
		a.problem(w, r, http.StatusNotFound, "DIRECTION_NOT_FOUND", "Direction was not found", "create_node", input.DirectionID, nil)
		return
	}
	if input.ParentID != nil {
		parent, err := a.nodeByID(r.Context(), *input.ParentID)
		if err == sql.ErrNoRows {
			a.problem(w, r, http.StatusUnprocessableEntity, "PARENT_NODE_NOT_FOUND", "Parent node was not found", "create_node", *input.ParentID, err)
			return
		}
		if err != nil {
			a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read parent node", "create_node", *input.ParentID, err)
			return
		}
		if parent.DirectionID != input.DirectionID {
			a.problem(w, r, http.StatusUnprocessableEntity, "PARENT_DIRECTION_MISMATCH", "Parent node belongs to another direction", "create_node", *input.ParentID, nil)
			return
		}
	}
	if err := normalizeDates(&input.TargetDate, &input.LastReviewedAt); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_ROADMAP_NODE", err.Error(), "create_node", 0, err)
		return
	}
	position := 0
	if input.Position != nil {
		position = *input.Position
	} else if err := a.db.QueryRowContext(r.Context(), `SELECT COALESCE(MAX(position) + 1, 0)
		FROM roadmap_nodes WHERE direction_id = ? AND parent_id IS ?`, input.DirectionID, input.ParentID).Scan(&position); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_CREATE_FAILED", "Could not create roadmap node", "create_node", 0, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `INSERT INTO roadmap_nodes(
		direction_id, parent_id, title, description, node_type, status, needs_review,
		confidence, position, next_action, target_date, last_reviewed_at, estimated_hours
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, input.DirectionID, input.ParentID,
		input.Title, input.Description, input.NodeType, input.Status, input.NeedsReview,
		input.Confidence, position, input.NextAction, input.TargetDate, input.LastReviewedAt, input.EstimatedHours)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_CREATE_FAILED", "Could not create roadmap node", "create_node", 0, err)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_CREATE_FAILED", "Could not create roadmap node", "create_node", 0, err)
		return
	}
	node, err := a.nodeByID(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read roadmap node", "create_node", id, err)
		return
	}
	writeJSON(w, http.StatusCreated, node)
}

func normalizeDates(targetDate, lastReviewedAt **string) error {
	if *targetDate != nil {
		if _, err := time.Parse(time.DateOnly, **targetDate); err != nil {
			return fmt.Errorf("target_date must use YYYY-MM-DD")
		}
	}
	if *lastReviewedAt != nil {
		value, err := time.Parse(time.RFC3339, **lastReviewedAt)
		if err != nil {
			return fmt.Errorf("last_reviewed_at must use RFC3339")
		}
		normalized := value.UTC().Format(time.RFC3339)
		*lastReviewedAt = &normalized
	}
	return nil
}

func (a *Handler) getNodeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_ROADMAP_NODE_ID", "Roadmap node id is invalid", "get_node", 0, err)
		return
	}
	node, err := a.nodeByID(r.Context(), id)
	if err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "ROADMAP_NODE_NOT_FOUND", "Roadmap node was not found", "get_node", id, err)
		return
	}
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read roadmap node", "get_node", id, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

type optionalString struct {
	Set   bool
	Value *string
}

func (value *optionalString) UnmarshalJSON(data []byte) error {
	value.Set = true
	if string(data) == "null" {
		value.Value = nil
		return nil
	}
	var decoded string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	value.Value = &decoded
	return nil
}

type nodeUpdate struct {
	Title          *string        `json:"title"`
	Description    *string        `json:"description"`
	NodeType       *string        `json:"node_type"`
	Status         *string        `json:"status"`
	NeedsReview    *bool          `json:"needs_review"`
	Confidence     *int           `json:"confidence"`
	Position       *int           `json:"position"`
	NextAction     *string        `json:"next_action"`
	TargetDate     optionalString `json:"target_date"`
	LastReviewedAt optionalString `json:"last_reviewed_at"`
	EstimatedHours *float64       `json:"estimated_hours"`
}

func (a *Handler) updateNode(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_ROADMAP_NODE_ID", "Roadmap node id is invalid", "update_node", 0, err)
		return
	}
	var input nodeUpdate
	if !a.decodeOrProblem(w, r, &input, "update_node") {
		return
	}
	node, err := a.nodeByID(r.Context(), id)
	if err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "ROADMAP_NODE_NOT_FOUND", "Roadmap node was not found", "update_node", id, err)
		return
	}
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read roadmap node", "update_node", id, err)
		return
	}
	if input.Title != nil {
		node.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		node.Description = *input.Description
	}
	if input.NodeType != nil {
		node.NodeType = *input.NodeType
	}
	if input.Status != nil {
		node.Status = *input.Status
	}
	if input.NeedsReview != nil {
		node.NeedsReview = *input.NeedsReview
	}
	if input.Confidence != nil {
		node.Confidence = *input.Confidence
	}
	if input.Position != nil {
		node.Position = *input.Position
	}
	if input.NextAction != nil {
		node.NextAction = *input.NextAction
	}
	if input.TargetDate.Set {
		node.TargetDate = input.TargetDate.Value
	}
	if input.LastReviewedAt.Set {
		node.LastReviewedAt = input.LastReviewedAt.Value
	}
	if input.EstimatedHours != nil {
		node.EstimatedHours = *input.EstimatedHours
	}
	if err := validateNode(node.Title, node.NodeType, node.Status, node.Confidence, node.EstimatedHours); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_ROADMAP_NODE", err.Error(), "update_node", id, err)
		return
	}
	if err := normalizeDates(&node.TargetDate, &node.LastReviewedAt); err != nil {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_ROADMAP_NODE", err.Error(), "update_node", id, err)
		return
	}
	_, err = a.db.ExecContext(r.Context(), `UPDATE roadmap_nodes SET title = ?, description = ?,
		node_type = ?, status = ?, needs_review = ?, confidence = ?, position = ?, next_action = ?,
		target_date = ?, last_reviewed_at = ?, estimated_hours = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`, node.Title,
		node.Description, node.NodeType, node.Status, node.NeedsReview, node.Confidence,
		node.Position, node.NextAction, node.TargetDate, node.LastReviewedAt, node.EstimatedHours, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_UPDATE_FAILED", "Could not update roadmap node", "update_node", id, err)
		return
	}
	node, err = a.nodeByID(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read roadmap node", "update_node", id, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (a *Handler) deleteNode(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_ROADMAP_NODE_ID", "Roadmap node id is invalid", "delete_node", 0, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `DELETE FROM roadmap_nodes WHERE id = ?`, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_DELETE_FAILED", "Could not delete roadmap node", "delete_node", id, err)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		a.problem(w, r, http.StatusNotFound, "ROADMAP_NODE_NOT_FOUND", "Roadmap node was not found", "delete_node", id, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type optionalInt64 struct {
	Set   bool
	Value *int64
}

func (value *optionalInt64) UnmarshalJSON(data []byte) error {
	value.Set = true
	if string(data) == "null" {
		value.Value = nil
		return nil
	}
	var decoded int64
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	value.Value = &decoded
	return nil
}

type nodeMove struct {
	ParentID optionalInt64 `json:"parent_id"`
	Position *int          `json:"position"`
}

func (a *Handler) moveNode(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_ROADMAP_NODE_ID", "Roadmap node id is invalid", "move_node", 0, err)
		return
	}
	var input nodeMove
	if !a.decodeOrProblem(w, r, &input, "move_node") {
		return
	}
	if !input.ParentID.Set && input.Position == nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_MOVE", "parent_id or position is required", "move_node", id, nil)
		return
	}
	node, err := a.nodeByID(r.Context(), id)
	if err == sql.ErrNoRows {
		a.problem(w, r, http.StatusNotFound, "ROADMAP_NODE_NOT_FOUND", "Roadmap node was not found", "move_node", id, err)
		return
	}
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read roadmap node", "move_node", id, err)
		return
	}
	parentID := node.ParentID
	if input.ParentID.Set {
		parentID = input.ParentID.Value
	}
	if parentID != nil {
		if *parentID <= 0 || *parentID == id {
			a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_MOVE", "A node cannot be its own parent", "move_node", id, nil)
			return
		}
		parent, err := a.nodeByID(r.Context(), *parentID)
		if err == sql.ErrNoRows {
			a.problem(w, r, http.StatusUnprocessableEntity, "PARENT_NODE_NOT_FOUND", "Parent node was not found", "move_node", *parentID, err)
			return
		}
		if err != nil {
			a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read parent node", "move_node", *parentID, err)
			return
		}
		if parent.DirectionID != node.DirectionID {
			a.problem(w, r, http.StatusUnprocessableEntity, "PARENT_DIRECTION_MISMATCH", "Parent node belongs to another direction", "move_node", *parentID, nil)
			return
		}
		var cycle bool
		err = a.db.QueryRowContext(r.Context(), `WITH RECURSIVE ancestors(id, parent_id) AS (
			SELECT id, parent_id FROM roadmap_nodes WHERE id = ?
			UNION ALL
			SELECT n.id, n.parent_id FROM roadmap_nodes n JOIN ancestors a ON n.id = a.parent_id
		) SELECT EXISTS(SELECT 1 FROM ancestors WHERE id = ?)`, *parentID, id).Scan(&cycle)
		if err != nil {
			a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_MOVE_FAILED", "Could not move roadmap node", "move_node", id, err)
			return
		}
		if cycle {
			a.problem(w, r, http.StatusUnprocessableEntity, "ROADMAP_CYCLE", "A node cannot be moved below its descendant", "move_node", id, nil)
			return
		}
	}
	position := node.Position
	if input.Position != nil {
		position = *input.Position
	}
	_, err = a.db.ExecContext(r.Context(), `UPDATE roadmap_nodes SET parent_id = ?, position = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`, parentID, position, id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_MOVE_FAILED", "Could not move roadmap node", "move_node", id, err)
		return
	}
	node, err = a.nodeByID(r.Context(), id)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read roadmap node", "move_node", id, err)
		return
	}
	writeJSON(w, http.StatusOK, node)
}

type dependencyCreate struct {
	DependsOnNodeID int64 `json:"depends_on_node_id"`
}

func (a *Handler) createDependency(w http.ResponseWriter, r *http.Request) {
	nodeID, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_ROADMAP_NODE_ID", "Roadmap node id is invalid", "create_dependency", 0, err)
		return
	}
	var input dependencyCreate
	if !a.decodeOrProblem(w, r, &input, "create_dependency") {
		return
	}
	if input.DependsOnNodeID <= 0 || input.DependsOnNodeID == nodeID {
		a.problem(w, r, http.StatusUnprocessableEntity, "INVALID_DEPENDENCY", "A node must depend on another existing node", "create_dependency", nodeID, nil)
		return
	}
	var count int
	if err := a.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM roadmap_nodes WHERE id IN (?, ?)`, nodeID, input.DependsOnNodeID).Scan(&count); err != nil {
		a.problem(w, r, http.StatusInternalServerError, "ROADMAP_NODE_READ_FAILED", "Could not read roadmap nodes", "create_dependency", nodeID, err)
		return
	}
	if count != 2 {
		a.problem(w, r, http.StatusNotFound, "ROADMAP_NODE_NOT_FOUND", "Roadmap node was not found", "create_dependency", nodeID, nil)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `INSERT INTO node_dependencies(node_id, depends_on_node_id)
		VALUES (?, ?) ON CONFLICT(node_id, depends_on_node_id) DO NOTHING`, nodeID, input.DependsOnNodeID)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DEPENDENCY_CREATE_FAILED", "Could not create dependency", "create_dependency", nodeID, err)
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		a.problem(w, r, http.StatusConflict, "DEPENDENCY_EXISTS", "Dependency already exists", "create_dependency", nodeID, nil)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DEPENDENCY_CREATE_FAILED", "Could not create dependency", "create_dependency", nodeID, err)
		return
	}
	var dependency NodeDependency
	err = a.db.QueryRowContext(r.Context(), `SELECT id, node_id, depends_on_node_id, created_at
		FROM node_dependencies WHERE id = ?`, id).Scan(&dependency.ID, &dependency.NodeID,
		&dependency.DependsOnNodeID, &dependency.CreatedAt)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DEPENDENCY_READ_FAILED", "Could not read dependency", "create_dependency", nodeID, err)
		return
	}
	writeJSON(w, http.StatusCreated, dependency)
}

func (a *Handler) deleteDependency(w http.ResponseWriter, r *http.Request) {
	nodeID, err := pathID(r, "id")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_ROADMAP_NODE_ID", "Roadmap node id is invalid", "delete_dependency", 0, err)
		return
	}
	dependencyID, err := pathID(r, "dependencyId")
	if err != nil {
		a.problem(w, r, http.StatusBadRequest, "INVALID_DEPENDENCY_ID", "Dependency id is invalid", "delete_dependency", nodeID, err)
		return
	}
	result, err := a.db.ExecContext(r.Context(), `DELETE FROM node_dependencies WHERE id = ? AND node_id = ?`, dependencyID, nodeID)
	if err != nil {
		a.problem(w, r, http.StatusInternalServerError, "DEPENDENCY_DELETE_FAILED", "Could not delete dependency", "delete_dependency", nodeID, err)
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		a.problem(w, r, http.StatusNotFound, "DEPENDENCY_NOT_FOUND", "Dependency was not found", "delete_dependency", nodeID, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

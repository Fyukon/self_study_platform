package app

import (
	"database/sql"
	"fmt"
	"strings"
)

type Settings struct {
	ID              int64  `json:"id"`
	ApplicationName string `json:"application_name"`
	Timezone        string `json:"timezone"`
	WeekStartsOn    int    `json:"week_starts_on"`
	DatabasePath    string `json:"database_path"`
	BackupPath      string `json:"backup_path"`
	Theme           string `json:"theme"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type Direction struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Position    int    `json:"position"`
	IsArchived  bool   `json:"is_archived"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type RoadmapNode struct {
	ID             int64            `json:"id"`
	DirectionID    int64            `json:"direction_id"`
	ParentID       *int64           `json:"parent_id"`
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	NodeType       string           `json:"node_type"`
	Status         string           `json:"status"`
	NeedsReview    bool             `json:"needs_review"`
	Confidence     int              `json:"confidence"`
	Position       int              `json:"position"`
	NextAction     string           `json:"next_action"`
	TargetDate     *string          `json:"target_date"`
	LastReviewedAt *string          `json:"last_reviewed_at"`
	EstimatedHours float64          `json:"estimated_hours"`
	CreatedAt      string           `json:"created_at"`
	UpdatedAt      string           `json:"updated_at"`
	Dependencies   []NodeDependency `json:"dependencies"`
	Children       []*RoadmapNode   `json:"children"`
}

type NodeDependency struct {
	ID              int64  `json:"id"`
	NodeID          int64  `json:"node_id"`
	DependsOnNodeID int64  `json:"depends_on_node_id"`
	CreatedAt       string `json:"created_at"`
}

var validStatuses = map[string]bool{
	"not_started": true,
	"learning":    true,
	"practicing":  true,
	"understood":  true,
}

var validNodeTypes = map[string]bool{
	"section":    true,
	"concept":    true,
	"practice":   true,
	"project":    true,
	"checkpoint": true,
}

func validateNode(title, nodeType, status string, confidence int, estimatedHours float64) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is required")
	}
	if !validNodeTypes[nodeType] {
		return fmt.Errorf("invalid node_type")
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status")
	}
	if confidence < 1 || confidence > 5 {
		return fmt.Errorf("confidence must be between 1 and 5")
	}
	if estimatedHours < 0 {
		return fmt.Errorf("estimated_hours cannot be negative")
	}
	return nil
}

const nodeColumns = `id, direction_id, parent_id, title, description, node_type, status,
	needs_review, confidence, position, next_action, target_date, last_reviewed_at,
	estimated_hours, created_at, updated_at`

type scanner interface {
	Scan(dest ...any) error
}

func scanNode(row scanner) (RoadmapNode, error) {
	var node RoadmapNode
	var parentID sql.NullInt64
	var targetDate, lastReviewedAt sql.NullString
	err := row.Scan(&node.ID, &node.DirectionID, &parentID, &node.Title, &node.Description,
		&node.NodeType, &node.Status, &node.NeedsReview, &node.Confidence, &node.Position,
		&node.NextAction, &targetDate, &lastReviewedAt, &node.EstimatedHours,
		&node.CreatedAt, &node.UpdatedAt)
	if err != nil {
		return RoadmapNode{}, err
	}
	if parentID.Valid {
		node.ParentID = &parentID.Int64
	}
	if targetDate.Valid {
		node.TargetDate = &targetDate.String
	}
	if lastReviewedAt.Valid {
		node.LastReviewedAt = &lastReviewedAt.String
	}
	node.Dependencies = []NodeDependency{}
	node.Children = []*RoadmapNode{}
	return node, nil
}

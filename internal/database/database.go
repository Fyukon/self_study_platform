package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"learning-roadmap/internal/config"
	"learning-roadmap/migrations"
	"learning-roadmap/seeds"
)

func Open(path string) (*sql.DB, error) {
	dsn := path
	if path != ":memory:" {
		u := url.URL{Scheme: "file", Path: path}
		dsn = u.String()
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	dsn += separator + "_busy_timeout=5000&_foreign_keys=on&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// ponytail: one connection is enough for the single-user MVP; raise it after measuring concurrent load.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return db, nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		var applied bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, entry.Name()).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}
		body, err := migrations.Files.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("start migration %s: %w", entry.Name(), err)
		}
		if _, err = tx.ExecContext(ctx, string(body)); err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES (?)`, entry.Name())
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func EnsureSettings(ctx context.Context, db *sql.DB, cfg config.Config) error {
	_, err := db.ExecContext(ctx, `INSERT INTO settings(
		id, application_name, timezone, week_starts_on, database_path, backup_path, theme
	) VALUES (1, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		cfg.ApplicationName, cfg.Timezone, cfg.WeekStartsOn, cfg.DatabasePath, cfg.BackupPath, cfg.Theme)
	if err != nil {
		return fmt.Errorf("create settings: %w", err)
	}
	return nil
}

type seedFile struct {
	SeedKey      string           `json:"seed_key"`
	Title        string           `json:"title"`
	Description  string           `json:"description"`
	Nodes        []seedNode       `json:"nodes"`
	Dependencies []seedDependency `json:"dependencies"`
}

type seedNode struct {
	SeedKey      string     `json:"seed_key"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	NodeType     string     `json:"node_type"`
	Status       string     `json:"status"`
	NeedsReview  bool       `json:"needs_review"`
	Confidence   int        `json:"confidence"`
	Position     int        `json:"position"`
	NextAction   string     `json:"next_action"`
	TargetDate   *string    `json:"target_date"`
	EstimatedHrs float64    `json:"estimated_hours"`
	Children     []seedNode `json:"children"`
}

type seedDependency struct {
	NodeSeedKey       string `json:"node_seed_key"`
	DependsOnSeedKey string `json:"depends_on_seed_key"`
}

func Seed(ctx context.Context, db *sql.DB) error {
	entries, err := seeds.Files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("list seeds: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if err := seedRoadmapFile(ctx, db, entry.Name()); err != nil {
			return fmt.Errorf("seed %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func seedRoadmapFile(ctx context.Context, db *sql.DB, name string) error {
	body, err := seeds.Files.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read seed: %w", err)
	}
	var seed seedFile
	if err := json.Unmarshal(body, &seed); err != nil {
		return fmt.Errorf("decode seed: %w", err)
	}
	if strings.TrimSpace(seed.SeedKey) == "" || strings.TrimSpace(seed.Title) == "" {
		return fmt.Errorf("validate seed: seed_key and title are required")
	}
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM directions WHERE seed_key = ?)`, seed.SeedKey).Scan(&exists); err != nil {
		return fmt.Errorf("check seed direction: %w", err)
	}
	if exists {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start seed transaction: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO directions(seed_key, title, description, icon, position)
		VALUES (?, ?, ?, 'server', 0) ON CONFLICT(seed_key) DO NOTHING`, seed.SeedKey, seed.Title, seed.Description); err != nil {
		return fmt.Errorf("insert seed direction: %w", err)
	}
	var directionID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM directions WHERE seed_key = ?`, seed.SeedKey).Scan(&directionID); err != nil {
		return fmt.Errorf("find seed direction: %w", err)
	}
	ids := make(map[string]int64)
	for i := range seed.Nodes {
		if err := insertSeedNode(ctx, tx, directionID, nil, seed.Nodes[i], ids); err != nil {
			return err
		}
	}
	for _, dependency := range seed.Dependencies {
		nodeID, ok := ids[dependency.NodeSeedKey]
		if !ok {
			return fmt.Errorf("validate seed dependency: unknown node %q", dependency.NodeSeedKey)
		}
		dependsOnID, ok := ids[dependency.DependsOnSeedKey]
		if !ok {
			return fmt.Errorf("validate seed dependency: unknown node %q", dependency.DependsOnSeedKey)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO node_dependencies(node_id, depends_on_node_id)
			VALUES (?, ?) ON CONFLICT(node_id, depends_on_node_id) DO NOTHING`, nodeID, dependsOnID); err != nil {
			return fmt.Errorf("insert seed dependency: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	return nil
}

func insertSeedNode(ctx context.Context, tx *sql.Tx, directionID int64, parentID *int64, node seedNode, ids map[string]int64) error {
	if strings.TrimSpace(node.SeedKey) == "" || strings.TrimSpace(node.Title) == "" {
		return fmt.Errorf("validate seed node: seed_key and title are required")
	}
	if _, exists := ids[node.SeedKey]; exists {
		return fmt.Errorf("validate seed node: duplicate seed_key %q", node.SeedKey)
	}
	if node.NodeType == "" {
		node.NodeType = "concept"
	}
	if node.Status == "" {
		node.Status = "not_started"
	}
	if node.Confidence == 0 {
		node.Confidence = 1
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO roadmap_nodes(
		seed_key, direction_id, parent_id, title, description, node_type, status,
		needs_review, confidence, position, next_action, target_date, estimated_hours
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(seed_key) DO NOTHING`, node.SeedKey, directionID, parentID, node.Title,
		node.Description, node.NodeType, node.Status, node.NeedsReview, node.Confidence,
		node.Position, node.NextAction, node.TargetDate, node.EstimatedHrs); err != nil {
		return fmt.Errorf("insert seed node %q: %w", node.SeedKey, err)
	}
	var id, actualDirectionID int64
	if err := tx.QueryRowContext(ctx, `SELECT id, direction_id FROM roadmap_nodes WHERE seed_key = ?`, node.SeedKey).Scan(&id, &actualDirectionID); err != nil {
		return fmt.Errorf("find seed node %q: %w", node.SeedKey, err)
	}
	if actualDirectionID != directionID {
		return fmt.Errorf("validate seed node %q: seed_key belongs to another direction", node.SeedKey)
	}
	ids[node.SeedKey] = id
	for i := range node.Children {
		if err := insertSeedNode(ctx, tx, directionID, &id, node.Children[i], ids); err != nil {
			return err
		}
	}
	return nil
}

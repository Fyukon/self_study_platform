package database

import (
	"context"
	"path/filepath"
	"testing"

	"learning-roadmap/internal/config"
)

func TestMigrateAndSeedAreIdempotent(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second migration run: %v", err)
	}
	cfg := config.Config{ApplicationName: "Learning Roadmap", Timezone: "Europe/Moscow", Theme: "system",
		WeekStartsOn: 1, DatabasePath: "test.db", BackupPath: "backups"}
	if err := EnsureSettings(ctx, db, cfg); err != nil {
		t.Fatal(err)
	}
	if err := Seed(ctx, db); err != nil {
		t.Fatal(err)
	}
	var directionsBefore, nodesBefore, dependenciesBefore int
	if err := db.QueryRow(`SELECT COUNT(*) FROM directions WHERE seed_key IS NOT NULL`).Scan(&directionsBefore); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM roadmap_nodes WHERE seed_key IS NOT NULL`).Scan(&nodesBefore); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM node_dependencies`).Scan(&dependenciesBefore); err != nil {
		t.Fatal(err)
	}
	if directionsBefore != 1 || nodesBefore == 0 || dependenciesBefore == 0 {
		t.Fatalf("incomplete seed: directions=%d nodes=%d dependencies=%d", directionsBefore, nodesBefore, dependenciesBefore)
	}
	if _, err := db.Exec(`DELETE FROM roadmap_nodes WHERE seed_key = 'backend-go.backend-foundations.dns'`); err != nil {
		t.Fatal(err)
	}
	if err := Seed(ctx, db); err != nil {
		t.Fatalf("second seed run: %v", err)
	}
	var directionsAfter, nodesAfter, dependenciesAfter int
	_ = db.QueryRow(`SELECT COUNT(*) FROM directions WHERE seed_key IS NOT NULL`).Scan(&directionsAfter)
	_ = db.QueryRow(`SELECT COUNT(*) FROM roadmap_nodes WHERE seed_key IS NOT NULL`).Scan(&nodesAfter)
	_ = db.QueryRow(`SELECT COUNT(*) FROM node_dependencies`).Scan(&dependenciesAfter)
	if directionsAfter != directionsBefore || nodesAfter != nodesBefore-1 || dependenciesAfter != dependenciesBefore {
		t.Fatalf("seed duplicated or restored user-deleted rows: before=(%d,%d,%d) after=(%d,%d,%d)",
			directionsBefore, nodesBefore, dependenciesBefore, directionsAfter, nodesAfter, dependenciesAfter)
	}
}

package migrations

import "embed"

// Files contains every immutable schema migration.
//
//go:embed *.sql
var Files embed.FS

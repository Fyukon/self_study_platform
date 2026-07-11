CREATE TABLE settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    application_name TEXT NOT NULL,
    timezone TEXT NOT NULL,
    week_starts_on INTEGER NOT NULL DEFAULT 1 CHECK (week_starts_on BETWEEN 1 AND 7),
    database_path TEXT NOT NULL,
    backup_path TEXT NOT NULL,
    theme TEXT NOT NULL DEFAULT 'system' CHECK (theme IN ('light', 'dark', 'system')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE directions (
    id INTEGER PRIMARY KEY,
    seed_key TEXT UNIQUE,
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    description TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0,
    is_archived INTEGER NOT NULL DEFAULT 0 CHECK (is_archived IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX directions_order_idx ON directions(is_archived, position, id);

CREATE TABLE roadmap_nodes (
    id INTEGER PRIMARY KEY,
    seed_key TEXT UNIQUE,
    direction_id INTEGER NOT NULL REFERENCES directions(id) ON DELETE CASCADE,
    parent_id INTEGER REFERENCES roadmap_nodes(id) ON DELETE CASCADE,
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    description TEXT NOT NULL DEFAULT '',
    node_type TEXT NOT NULL DEFAULT 'concept' CHECK (node_type IN ('section', 'concept', 'practice', 'project', 'checkpoint')),
    status TEXT NOT NULL DEFAULT 'not_started' CHECK (status IN ('not_started', 'learning', 'practicing', 'understood')),
    needs_review INTEGER NOT NULL DEFAULT 0 CHECK (needs_review IN (0, 1)),
    confidence INTEGER NOT NULL DEFAULT 1 CHECK (confidence BETWEEN 1 AND 5),
    position INTEGER NOT NULL DEFAULT 0,
    next_action TEXT NOT NULL DEFAULT '',
    target_date TEXT,
    last_reviewed_at TEXT,
    estimated_hours REAL NOT NULL DEFAULT 0 CHECK (estimated_hours >= 0),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX roadmap_nodes_tree_idx ON roadmap_nodes(direction_id, parent_id, position, id);
CREATE INDEX roadmap_nodes_status_idx ON roadmap_nodes(direction_id, status, needs_review);

CREATE TABLE node_dependencies (
    id INTEGER PRIMARY KEY,
    node_id INTEGER NOT NULL REFERENCES roadmap_nodes(id) ON DELETE CASCADE,
    depends_on_node_id INTEGER NOT NULL REFERENCES roadmap_nodes(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (node_id <> depends_on_node_id),
    UNIQUE (node_id, depends_on_node_id)
);

CREATE INDEX node_dependencies_target_idx ON node_dependencies(depends_on_node_id);

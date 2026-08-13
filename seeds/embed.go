package seeds

import "embed"

// Files contains roadmap data imported on the first application start.
// Each JSON file is an independent idempotent roadmap seed.
//
//go:embed *.json
var Files embed.FS

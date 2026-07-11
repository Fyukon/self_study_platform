package seeds

import "embed"

// Files contains data imported on the first application start.
//
//go:embed backend_go_roadmap.json
var Files embed.FS

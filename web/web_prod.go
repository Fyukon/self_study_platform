//go:build production

package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var assets embed.FS

// Handler serves Vite assets and falls back to index.html for client-side routes.
func Handler() http.Handler {
	root, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		info, statErr := fs.Stat(root, name)
		if name == "." || statErr != nil || info.IsDir() {
			request := r.Clone(r.Context())
			request.URL.Path = "/"
			files.ServeHTTP(w, request)
			return
		}
		files.ServeHTTP(w, r)
	})
}

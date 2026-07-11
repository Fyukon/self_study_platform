//go:build !production

package web

import "net/http"

// Handler points developers to Vite; production builds replace it with embedded assets.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "frontend is available at http://127.0.0.1:5173 during development", http.StatusServiceUnavailable)
	})
}

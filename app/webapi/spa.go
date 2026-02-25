package webapi

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed frontend/dist
var spaFS embed.FS

// spaHandler serves the React SPA. It serves static files from frontend/dist
// and falls back to index.html for client-side routing.
func spaHandler() http.Handler {
	distFS, err := fs.Sub(spaFS, "frontend/dist")
	if err != nil {
		panic("failed to get frontend/dist subtree: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// skip API routes
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// try to serve a static file first
		if path != "/" {
			cleanPath := strings.TrimPrefix(path, "/")
			if f, err := distFS.Open(cleanPath); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// fall back to index.html for client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

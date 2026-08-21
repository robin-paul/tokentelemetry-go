package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves embedded static assets with SPA fallback.
func Handler() http.Handler {
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(subFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := path.Clean(r.URL.Path)

		// 1. Static asset caching
		if strings.HasPrefix(reqPath, "/_astro/") || strings.HasPrefix(reqPath, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			fileServer.ServeHTTP(w, r)
			return
		}

		// 2. Check if file exists directly in embed.FS
		f, err := subFS.Open(strings.TrimPrefix(reqPath, "/"))
		if err == nil {
			_ = f.Close()
			w.Header().Set("Cache-Control", "no-cache")
			fileServer.ServeHTTP(w, r)
			return
		}

		// 3. Dynamic route fallbacks for React client islands
		if strings.HasPrefix(reqPath, "/sessions/") {
			r.URL.Path = "/sessions/[id]/index.html"
		} else if strings.HasPrefix(reqPath, "/projects/") {
			r.URL.Path = "/projects/[...path]/index.html"
		} else {
			// Default 404 / root fallback
			r.URL.Path = "/index.html"
		}

		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})
}

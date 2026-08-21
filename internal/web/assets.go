package web

import (
	"embed"
	"io"
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

		// 2. Check if file or directory exists directly in embed.FS
		cleanPath := strings.TrimPrefix(reqPath, "/")
		if cleanPath == "" {
			cleanPath = "."
		}

		f, err := subFS.Open(cleanPath)
		if err == nil {
			stat, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil {
				if stat.IsDir() {
					indexPath := path.Join(cleanPath, "index.html")
					if idxF, idxErr := subFS.Open(indexPath); idxErr == nil {
						_ = idxF.Close()
						w.Header().Set("Cache-Control", "no-cache")
						fileServer.ServeHTTP(w, r)
						return
					}
				} else {
					w.Header().Set("Cache-Control", "no-cache")
					fileServer.ServeHTTP(w, r)
					return
				}
			}
		}

		// 3. Dynamic route fallbacks for React client islands
		fallbackFile := "index.html"
		if strings.HasPrefix(reqPath, "/sessions/") {
			fallbackFile = "sessions/[id]/index.html"
		} else if strings.HasPrefix(reqPath, "/projects/") {
			fallbackFile = "projects/[...path]/index.html"
		}

		fb, err := subFS.Open(fallbackFile)
		if err != nil {
			fb, err = subFS.Open("index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}
		defer fb.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, fb)
	})
}

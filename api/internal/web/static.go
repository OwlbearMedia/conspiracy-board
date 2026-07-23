// Package web serves the embedded frontend bundle in preview/single-container
// deploys. The dist directory is populated by the Docker build (`vite build`
// output copied in); locally it contains only a placeholder.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// SPAHandler serves static files from the embedded dist, falling back to
// index.html for client-side routes.
func SPAHandler() http.HandlerFunc {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err) // embed is broken at build time, not recoverable
	}
	fileServer := http.FileServer(http.FS(sub))

	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if f, err := sub.Open(path); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		r.URL.Path = "/" // SPA fallback
		fileServer.ServeHTTP(w, r)
	}
}

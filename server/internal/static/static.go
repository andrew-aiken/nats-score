// Package static serves the frontend's production build, embedded into the
// server binary at compile time. The dist/ directory is populated by
// building the frontend (npm run build) and copying its output here before
// running go build.
package static

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler serves the embedded frontend build. Requests for paths that don't
// match a file in the build (e.g. client-side routes like /settings) fall
// back to index.html so the SPA's router can take over.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, fmt.Errorf("create static sub filesystem: %w", err)
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		if _, err := fs.Stat(sub, path); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	}), nil
}

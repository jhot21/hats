// internal/admin/admin.go
package admin

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"codeberg.org/jhot/hats/internal/admin/templates"
	"codeberg.org/jhot/hats/pkg/config"
	"github.com/go-chi/chi/v5"
)

//go:embed static
var staticFiles embed.FS

// Init mounts the /admin sub-router onto the provided Chi router.
func Init(r chi.Router, cfg *config.HatsConfig, logger *slog.Logger) {
	adminCfg = cfg
	adminLogger = logger
	initSessionKey()

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic("admin: failed to create static sub-FS: " + err.Error())
	}

	r.Route("/admin", func(r chi.Router) {
		r.Get("/login", loginHandler)
		r.Post("/login", loginHandler)
		r.Get("/logout", logoutHandler)
		r.Handle("/static/*", http.StripPrefix("/admin/static/", http.FileServer(http.FS(staticFS))))

		r.Group(func(r chi.Router) {
			r.Use(requireAuth)
			r.Get("/", dashboardHandler)
			r.Get("/api/status", statusHandler)
		})
	})
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	templates.Dashboard().Render(r.Context(), w)
}

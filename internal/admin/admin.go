// internal/admin/admin.go
package admin

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/jhot21/hats/internal/admin/templates"
	"github.com/jhot21/hats/internal/cache"
	"github.com/jhot21/hats/internal/cmdstats"
	"github.com/jhot21/hats/internal/secrets"
	"github.com/jhot21/hats/internal/serverconfig"
	"github.com/jhot21/hats/internal/tokens"
	"github.com/go-chi/chi/v5"
)

//go:embed static
var staticFiles embed.FS

var adminTokenStore tokens.TokenStore
var adminSecretStore secrets.SecretStore
var adminCacheStats *cache.Stats
var adminMirror *cache.EntityMirror
var adminCmdStats *cmdstats.Recorder

// Init mounts the /admin sub-router onto the provided Chi router.
func Init(r chi.Router, cfg *serverconfig.ServerConfig, logger *slog.Logger, ts tokens.TokenStore, ss secrets.SecretStore, cs *cache.Stats, mirror *cache.EntityMirror, crs *cmdstats.Recorder) {
	adminCfg = cfg
	adminLogger = logger
	adminTokenStore = ts
	adminSecretStore = ss
	adminCacheStats = cs
	adminMirror = mirror
	adminCmdStats = crs
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

			// Token management pages
			r.Get("/tokens", tokenListHandler)
			r.Get("/tokens/new", tokenNewHandler)
			r.Get("/tokens/table", tokenTableHandler)
			r.Get("/tokens/{hash}/edit", tokenEditHandler)

			// Token management API
			r.Get("/api/tokens", tokenJSONListHandler)
			r.Post("/api/tokens", tokenCreateAPIHandler)
			r.Post("/api/tokens/{hash}/revoke", tokenRevokeAPIHandler)
			r.Put("/api/tokens/{hash}", tokenUpdateAPIHandler)

			// Secret management pages
			r.Get("/secrets", secretListHandler)
			r.Get("/secrets/new", secretNewHandler)
			r.Get("/secrets/table", secretTableHandler)
			r.Get("/secrets/{name}/edit", secretEditHandler)

			// Secret management API
			r.Get("/api/secrets", secretJSONListHandler)
			r.Post("/api/secrets", secretCreateAPIHandler)
			r.Put("/api/secrets/{name}", secretUpdateAPIHandler)
			r.Delete("/api/secrets/{name}", secretDeleteAPIHandler)

			// Home Assistant connection management
			r.Get("/homeassistant", haPageHandler)
			r.Post("/api/homeassistant", haSaveHandler)
			r.Post("/api/homeassistant/test", haTestHandler)
			r.Get("/api/homeassistant/token", haTokenHandler)
			r.Post("/api/restart", restartHandler)

			// ntfy connection management
			r.Get("/ntfy", ntfyPageHandler)
			r.Post("/api/ntfy", ntfySaveHandler)
			r.Post("/api/ntfy/test", ntfyTestHandler)
			r.Get("/api/ntfy/token", ntfyTokenHandler)

			// Cache browser
			r.Get("/cache", cachePageHandler)
			r.Get("/api/cache/stats", cacheStatsHandler)
			r.Get("/api/cache/entities", cacheEntitiesHandler)
			r.Post("/api/cache/stats/reset", cacheStatsResetHandler)

			// Command browser
			r.Get("/commands", commandBrowserHandler)
			r.Get("/api/commands/stats", commandStatsHandler)
			r.Get("/api/commands/dashboard", commandDashboardStatsHandler)
			r.Get("/api/commands/tokens", commandTokensHandler)
		})
	})
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	templates.Dashboard().Render(r.Context(), w)
}

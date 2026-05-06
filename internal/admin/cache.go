package admin

import (
	"net/http"

	"codeberg.org/jhot/hats/internal/admin/templates"
	"codeberg.org/jhot/hats/internal/cache"
)

func cachePageHandler(w http.ResponseWriter, r *http.Request) {
	templates.Cache().Render(r.Context(), w)
}

func cacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	snap := adminCacheStats.Snapshot()

	// Merge key counts from KV into domain stats
	if adminHatsKV != nil {
		entries, _ := cache.ListEntries(r.Context(), adminHatsKV, "", "")
		keyCounts := map[string]int{}
		for _, e := range entries {
			keyCounts[e.Domain]++
		}
		for i := range snap {
			snap[i].Keys = keyCounts[snap[i].Domain]
		}
	}

	data := templates.CacheStatsData{Domains: snap}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CacheStats(data).Render(r.Context(), w)
}

func cacheEntitiesHandler(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	q := r.URL.Query().Get("q")

	var entries []cache.CacheEntry
	if adminHatsKV != nil {
		var err error
		entries, err = cache.ListEntries(r.Context(), adminHatsKV, domain, q)
		if err != nil {
			adminLogger.Error("cache browser: ListEntries failed", "error", err)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CacheEntities(entries).Render(r.Context(), w)
}

func cacheStatsResetHandler(w http.ResponseWriter, r *http.Request) {
	if err := adminCacheStats.Reset(r.Context()); err != nil {
		adminLogger.Error("cache stats reset failed", "error", err)
	}
	// Return updated stats fragment
	cacheStatsHandler(w, r)
}

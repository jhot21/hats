package admin

import (
	"net/http"

	"github.com/jhot21/hats/internal/admin/templates"
	"github.com/jhot21/hats/internal/cmdstats"
)

func commandBrowserHandler(w http.ResponseWriter, r *http.Request) {
	stats, summary := adminCmdStats.Snapshot()
	endpoints := templates.GroupByEndpoint(stats)
	data := templates.CommandBrowserData{
		Endpoints: endpoints,
		Summary:   summary,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CommandBrowser(data).Render(r.Context(), w)
}

func commandStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, summary := adminCmdStats.Snapshot()
	endpoints := templates.GroupByEndpoint(stats)
	data := templates.CommandBrowserData{
		Endpoints: endpoints,
		Summary:   summary,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CommandBrowserList(data).Render(r.Context(), w)
}

func commandDashboardStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, summary := adminCmdStats.Snapshot()
	endpoints := templates.GroupByEndpoint(stats)
	data := templates.CommandStatsData{
		Endpoints: endpoints,
		Summary:   summary,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CommandStats(data).Render(r.Context(), w)
}

func commandTokensHandler(w http.ResponseWriter, r *http.Request) {
	cmdName := r.URL.Query().Get("cmd")
	allStats, _ := adminCmdStats.Snapshot()

	var tokenStats []cmdstats.CommandStat
	for _, s := range allStats {
		if s.CommandName == cmdName {
			tokenStats = append(tokenStats, s)
		}
	}

	data := templates.CommandTokensData{
		CommandName: cmdName,
		Tokens:      tokenStats,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.CommandTokens(data).Render(r.Context(), w)
}

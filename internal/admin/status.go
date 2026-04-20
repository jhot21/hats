// internal/admin/status.go
package admin

import (
	"net/http"
	"time"

	"codeberg.org/jhot/hats/internal/admin/templates"
	"codeberg.org/jhot/hats/internal/homeassistant"
	"codeberg.org/jhot/hats/internal/nats"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	checkedAt := "Checked at " + time.Now().Format("15:04:05")
	services := []templates.ServiceStatus{
		{Name: "Home Assistant", Connected: homeassistant.IsConnected(), CheckedAt: checkedAt},
		{Name: "NATS", Connected: nats.IsConnected(), CheckedAt: checkedAt},
		{Name: "API", Connected: true, CheckedAt: checkedAt},
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.StatusCards(services).Render(r.Context(), w)
}

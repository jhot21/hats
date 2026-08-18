// internal/admin/status.go
package admin

import (
	"net/http"
	"time"

	"github.com/jhot21/hats/internal/admin/templates"
	"github.com/jhot21/hats/internal/homeassistant"
	"github.com/jhot21/hats/internal/nats"
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

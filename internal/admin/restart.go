package admin

import (
	"net/http"
	"os"
	"time"

	"codeberg.org/jhot/hats/internal/admin/templates"
)

func restartHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.HARestartResult().Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render response", http.StatusInternalServerError)
		return
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Exit(0)
	}()
}

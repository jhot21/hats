package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/jhot21/hats/integration/haserver"
)

func main() {
	token := os.Getenv("MOCK_HA_TOKEN")
	if token == "" {
		token = "test-token"
	}
	portStr := os.Getenv("MOCK_HA_PORT")
	if portStr == "" {
		portStr = "8123"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("invalid MOCK_HA_PORT: %v", err)
	}

	// NewBare creates the server without starting httptest.Server so we can
	// host everything — WebSocket and admin endpoints — on a single known port.
	server := haserver.NewBare(token)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux) // registers /api/websocket

	// Admin HTTP endpoints for event injection (used by E2E test scripts)
	mux.HandleFunc("/inject/state", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			EntityId string `json:"entity_id"`
			OldState string `json:"old_state"`
			NewState string `json:"new_state"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		server.InjectStateChange(req.EntityId, req.OldState, req.NewState)
		w.WriteHeader(204)
	})
	mux.HandleFunc("/inject/zha", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DeviceIeee string `json:"device_ieee"`
			Command    string `json:"command"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		server.InjectZHAEvent(req.DeviceIeee, req.Command)
		w.WriteHeader(204)
	})
	mux.HandleFunc("/inject/nfc", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ TagId string `json:"tag_id"` }
		json.NewDecoder(r.Body).Decode(&req)
		server.InjectNFCEvent(req.TagId)
		w.WriteHeader(204)
	})
	mux.HandleFunc("/inject/timer", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ EntityId string `json:"entity_id"` }
		json.NewDecoder(r.Body).Decode(&req)
		server.InjectTimerFinished(req.EntityId)
		w.WriteHeader(204)
	})
	mux.HandleFunc("/inject/zwave", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ DeviceId string `json:"device_id"` }
		json.NewDecoder(r.Body).Decode(&req)
		server.InjectZWaveEvent(req.DeviceId)
		w.WriteHeader(204)
	})
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{
			"connections": server.ConnectionCount(),
			"pings":       server.PingsReceived(),
		})
	})

	addr := fmt.Sprintf(":%d", port)
	slog.Info("mockha listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

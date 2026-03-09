package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"codeberg.org/jhot/hats/integration/haserver"
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

	server := haserver.New(token)

	// Admin HTTP endpoints for event injection (used by E2E test scripts)
	http.HandleFunc("/inject/state", func(w http.ResponseWriter, r *http.Request) {
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
	http.HandleFunc("/inject/zha", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DeviceIeee string `json:"device_ieee"`
			Command    string `json:"command"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		server.InjectZHAEvent(req.DeviceIeee, req.Command)
		w.WriteHeader(204)
	})
	http.HandleFunc("/inject/nfc", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ TagId string `json:"tag_id"` }
		json.NewDecoder(r.Body).Decode(&req)
		server.InjectNFCEvent(req.TagId)
		w.WriteHeader(204)
	})
	http.HandleFunc("/inject/timer", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ EntityId string `json:"entity_id"` }
		json.NewDecoder(r.Body).Decode(&req)
		server.InjectTimerFinished(req.EntityId)
		w.WriteHeader(204)
	})
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{
			"connections": server.ConnectionCount(),
			"pings":       server.PingsReceived(),
		})
	})

	slog.Info("mockha listening", "port", port, "ha_port", server.Addr())
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal(err)
	}
}

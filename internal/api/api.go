package api

import (
	"fmt"
	"net/http"
	"time"

	"log/slog"

	"code.jhot.me/jhot/hats/internal/nats"
	"code.jhot.me/jhot/hats/pkg/config"
	"code.jhot.me/jhot/hats/pkg/homeassistant"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

var (
	cfg      *config.HatsConfig
	logger   *slog.Logger
	server   http.Server
	haClient *homeassistant.RestClient
)

const (
	HA_STATE_PREFIX = "homeassistant.states"
)

func Listen(parentLogger *slog.Logger) {
	logger = parentLogger
	cfg = config.FromEnvironment()
	haClient = homeassistant.NewRestClient(cfg.HomeAssistantBaseUrl, cfg.HomeAssistantToken)
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	router.Get(`/api/state/{entityId}`, func(w http.ResponseWriter, r *http.Request) {
		logger.Debug(fmt.Sprintf("%s %s", r.Method, r.URL.Path), "method", r.Method, "path", r.URL.Path, "address", r.RemoteAddr)
		entityId := chi.URLParam(r, "entityId")

		kvVal, err := nats.GetKeyValue(fmt.Sprintf("%s.%s", HA_STATE_PREFIX, entityId))
		if err == nil && len(kvVal) > 0 {
			w.Write(kvVal)
			return
		}

		data, err := haClient.GetState(entityId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		nats.SetKeyValueString(fmt.Sprintf("%s.%s", HA_STATE_PREFIX, entityId), data.State)
		render.PlainText(w, r, data.State)
	})

	router.Post("/api/state/{entityId}/{service}", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug(fmt.Sprintf("%s %s", r.Method, r.URL.Path), "method", r.Method, "path", r.URL.Path, "address", r.RemoteAddr)
		entityId := chi.URLParam(r, "entityId")
		service := chi.URLParam(r, "service")

		var extras map[string]string
		err := render.DecodeJSON(r.Body, &extras)
		var haErr error
		if err == nil && len(extras) > 0 {
			haErr = haClient.CallService(entityId, service, extras)
		} else {
			haErr = haClient.CallService(entityId, service)
		}

		if haErr != nil {
			logger.Error("Error setting state", "error", haErr)
			http.Error(w, fmt.Sprintf("error proxying request: %s", haErr.Error()), http.StatusInternalServerError)
			return
		}

		render.Status(r, http.StatusOK)
		render.PlainText(w, r, "OK")
	})

	server = http.Server{
		Addr:    ":8888",
		Handler: router,
	}

	go server.ListenAndServe()
}

func Close() {
	if server.Addr != "" {
		server.Close()
	}
}

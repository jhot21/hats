package api

import (
	"fmt"
	"net/http"
	"time"

	"log/slog"

	"code.jhot.me/jhot/hats/internal/nats"
	"code.jhot.me/jhot/hats/internal/ntfy"
	"code.jhot.me/jhot/hats/pkg/config"
	"code.jhot.me/jhot/hats/pkg/homeassistant"
	ntfyPkg "code.jhot.me/jhot/hats/pkg/ntfy"
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
	haClient = homeassistant.NewRestClient(cfg.GetHomeAssistantBaseUrl(), cfg.HomeAssistantToken)
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	router.Get(`/api/state/{entityId}`, getEntityStateHandler)
	router.Post("/api/state/{entityId}/{service}", setEntityStateHandler)

	router.Get("/api/timer/{timerName}", getTimerHandler)
	router.Post("/api/timer/{timerName}", startTimerHandler)
	router.Delete("/api/timer/{timerName}", deleteTimerHandler)

	router.Get("/api/schedule/{scheduleName}", getScheduleHandler)
	router.Post("/api/schedule/{scheduleName}", createScheduleHandler)
	router.Delete("/api/schedule/{scheduleName}", deleteScheduleHandler)

	router.Post("/api/ntfy", postNtfyHandler)

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

func logRequest(w http.ResponseWriter, r *http.Request) {
	logger.Debug(fmt.Sprintf("%s %s", r.Method, r.URL.Path), "method", r.Method, "path", r.URL.Path, "address", r.RemoteAddr)
}

// HOME ASSISTANT ENTITIES

func getEntityStateHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(w, r)
	entityId := chi.URLParam(r, "entityId")
	full := r.URL.Query().Get("full") == "true"

	if !full {
		kvVal, err := nats.GetKeyValue(fmt.Sprintf("%s.%s", HA_STATE_PREFIX, entityId))
		if err == nil && len(kvVal) > 0 {
			w.Write(kvVal)
			return
		}
	}

	data, err := haClient.GetState(entityId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nats.SetKeyValueString(fmt.Sprintf("%s.%s", HA_STATE_PREFIX, entityId), data.State)
	if full {
		render.JSON(w, r, data)
	} else {
		render.PlainText(w, r, data.State)
	}
}

func setEntityStateHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(w, r)
	entityId := chi.URLParam(r, "entityId")
	service := chi.URLParam(r, "service")
	domain := r.URL.Query().Get("domain")

	var extras map[string]any
	err := render.DecodeJSON(r.Body, &extras)
	var haErr error
	if err == nil && len(extras) > 0 {
		if domain != "" {
			haErr = haClient.CallServiceManual(domain, entityId, service, extras)
		} else {
			haErr = haClient.CallService(entityId, service, extras)
		}
	} else {
		if domain != "" {
			haErr = haClient.CallServiceManual(domain, entityId, service)
		} else {
			haErr = haClient.CallService(entityId, service)
		}
	}

	if haErr != nil {
		logger.Error("Error setting state", "error", haErr)
		http.Error(w, fmt.Sprintf("error proxying request: %s", haErr.Error()), http.StatusInternalServerError)
		return
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "OK")
}

// TIMERS

func getTimerHandler(w http.ResponseWriter, r *http.Request) {
	timerName := chi.URLParam(r, "timerName")
	logRequest(w, r)

	timer, err := nats.GetTimer(timerName)
	if err != nil {
		http.Error(w, "Unable to get timer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	render.PlainText(w, r, timer.ToString())
}

type StartTimerData struct {
	Duration string `json:"duration"`
	Force    bool   `json:"force"`
}

func startTimerHandler(w http.ResponseWriter, r *http.Request) {
	timerName := chi.URLParam(r, "timerName")
	logRequest(w, r)

	data := &StartTimerData{}
	if err := render.DecodeJSON(r.Body, data); err != nil {
		http.Error(w, "Unable to parse timer data", http.StatusNotAcceptable)
		return
	}

	timer, err := nats.GetTimer(timerName)
	if err != nil {
		http.Error(w, "Unable to get timer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if data.Duration == "" {
		data.Duration = timer.Duration.String()
	}

	if data.Force {
		timer.Activate(data.Duration)
	} else {
		timer.ActivateIfNotAlready(data.Duration)
	}

	getTimerHandler(w, r)
}

func deleteTimerHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(w, r)
	timerName := chi.URLParam(r, "timerName")

	timer, err := nats.GetTimer(timerName)
	if err != nil {
		http.Error(w, "Unable to get timer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	timer.Cancel()
	render.PlainText(w, r, "OK")
}

// SCHEDULES

func getScheduleHandler(w http.ResponseWriter, r *http.Request) {
	scheduleName := chi.URLParam(r, "scheduleName")
	logRequest(w, r)

	schedule, err := nats.GetSchedule(scheduleName)
	if err != nil {
		http.Error(w, "Unable to get schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	render.PlainText(w, r, string(schedule.GetNext()))
}

type CreateScheduleData struct {
	Cron string `json:"cron"`
}

func createScheduleHandler(w http.ResponseWriter, r *http.Request) {
	scheduleName := chi.URLParam(r, "scheduleName")
	logRequest(w, r)

	data := &CreateScheduleData{}
	if err := render.DecodeJSON(r.Body, data); err != nil {
		http.Error(w, "Unable to parse schedule data", http.StatusNotAcceptable)
		return
	}

	if data.Cron == "" {
		http.Error(w, "cron required", http.StatusNotAcceptable)
		return
	}

	schedule := nats.NewSchedule(scheduleName, data.Cron)
	schedule.Activate()

	getScheduleHandler(w, r)
}

func deleteScheduleHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(w, r)
	scheduleName := chi.URLParam(r, "scheduleName")

	schedule, err := nats.GetSchedule(scheduleName)
	if err != nil {
		http.Error(w, "Unable to get schedule: "+err.Error(), http.StatusInternalServerError)
		return
	}

	schedule.Cancel()
	render.PlainText(w, r, "OK")
}

// NTFY

func postNtfyHandler(w http.ResponseWriter, r *http.Request) {
	data := &ntfyPkg.Message{}
	if err := render.DecodeJSON(r.Body, data); err != nil {
		http.Error(w, "Unable to parse message data", http.StatusNotAcceptable)
		return
	}

	if err := ntfy.Send(*data); err != nil {
		http.Error(w, "Unable to send message", http.StatusBadRequest)
		return
	}

	render.PlainText(w, r, "OK")
}

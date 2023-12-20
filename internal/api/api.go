package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
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
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

var (
	cfg      *config.HatsConfig
	logger   *slog.Logger
	server   http.Server
	haClient *homeassistant.RestClient
	homepage string
)

const (
	HA_STATE_PREFIX = "homeassistant.states"
)

func Listen(parentLogger *slog.Logger, parentConfig *config.HatsConfig, readme []byte) {
	logger = parentLogger
	cfg = parentConfig

	// render readme to HTML
	p := parser.NewWithExtensions(parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock)
	doc := p.Parse(readme)
	homepage = string(markdown.Render(doc, html.NewRenderer(html.RendererOptions{Flags: html.CommonFlags | html.HrefTargetBlank})))

	haClient = homeassistant.NewRestClient(cfg.GetHomeAssistantBaseUrl(), cfg.HomeAssistantToken)
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(loggerMiddleware)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		render.HTML(w, r, homepage)
	})
	router.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "OK")
	})

	router.Route("/api", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get(`/state/{entityId}`, getEntityStateHandler)
		r.Post("/state/{entityId}/{service}", setEntityStateHandler)

		r.Get("/timer/{timerName}", getTimerHandler)
		r.Post("/timer/{timerName}", startTimerHandler)
		r.Delete("/timer/{timerName}", deleteTimerHandler)

		r.Get("/schedule/{scheduleName}", getScheduleHandler)
		r.Post("/schedule/{scheduleName}", createScheduleHandler)
		r.Delete("/schedule/{scheduleName}", deleteScheduleHandler)

		r.Post("/ntfy", postNtfyHandler)

		r.Post("/command/{commandName}", postCommandHandler)
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

// authMiddleware checks both basic and bearer auth schemes for a token
//
//	When using basic auth: the username does not matter and the the password should equal the configured token
//	When using bearer auth: set the "Authorization" header to "Bearer your-token-here"
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.HatsToken == "" { // No token required
			next.ServeHTTP(w, r)
			return
		}

		_, p, ok := r.BasicAuth()

		if ok && p == cfg.HatsToken {
			next.ServeHTTP(w, r)
			return
		}

		authHeaderPars := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(authHeaderPars) > 1 && strings.EqualFold(authHeaderPars[0], "bearer") && authHeaderPars[1] == cfg.HatsToken {
			next.ServeHTTP(w, r)
			return
		}

		logger.Warn("Unauthorized request", "method", r.Method, "path", r.URL.Path, "address", r.RemoteAddr)
		http.Error(w, "Bearer authorization header doesn't match configured token", http.StatusUnauthorized)
	})
}

func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug(fmt.Sprintf("%s %s", r.Method, r.URL.Path), "method", r.Method, "path", r.URL.Path, "address", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// HOME ASSISTANT ENTITIES

func getEntityStateHandler(w http.ResponseWriter, r *http.Request) {
	entityId := chi.URLParam(r, "entityId")
	full := r.URL.Query().Get("full") == "true"
	l := logger.With("endpoint", "GET /api/state/{entityId}", "entityId", entityId, "full", full)

	if !full {
		l.Debug("Getting state from KV store")
		kvVal, err := nats.GetKeyValue(fmt.Sprintf("%s.%s", HA_STATE_PREFIX, entityId))
		if err == nil && len(kvVal) > 0 {
			l.Debug("Returning", "value", string(kvVal))
			render.PlainText(w, r, string(kvVal))
			return
		}
	}

	l.Debug("Getting state from Home Assistant")
	data, err := haClient.GetState(entityId)
	if err != nil {
		l.Error("Error getting state from Home Assistant", "error", err)
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
	entityId := chi.URLParam(r, "entityId")
	service := chi.URLParam(r, "service")
	domain := r.URL.Query().Get("domain")

	l := logger.With("endpoint", "POST /api/state/{entityId}/{service}", "entityId", entityId, "service", service, "domain", domain)

	var extras map[string]any
	err := render.DecodeJSON(r.Body, &extras)
	if err != nil {
		b, _ := io.ReadAll(r.Body)
		l.Error("Error decoding JSON body", "error", err, "body", string(b))
	}
	var haErr error
	if err == nil && len(extras) > 0 {
		if domain != "" {
			l.Debug("Calling service manually", "extras", extras)
			haErr = haClient.CallServiceManual(domain, entityId, service, extras)
		} else {
			l.Debug("Calling service", "extras", extras)
			haErr = haClient.CallService(entityId, service, extras)
		}
	} else {
		if domain != "" {
			l.Debug("Calling service manually (without extras)")
			haErr = haClient.CallServiceManual(domain, entityId, service)
		} else {
			l.Debug("Calling service (without extras)")
			haErr = haClient.CallService(entityId, service)
		}
	}

	if haErr != nil {
		l.Error("Error setting state", "error", haErr)
		http.Error(w, fmt.Sprintf("error proxying request: %s", haErr.Error()), http.StatusInternalServerError)
		return
	}

	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "OK")
}

// TIMERS

func getTimerHandler(w http.ResponseWriter, r *http.Request) {
	timerName := chi.URLParam(r, "timerName")

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

// Command

func postCommandHandler(w http.ResponseWriter, r *http.Request) {
	commandName := chi.URLParam(r, "commandName")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Error reading request body", "error", err, "url", r.URL.String())
	}
	nats.Publish(fmt.Sprintf("command.%s", commandName), body)
	render.PlainText(w, r, "OK")
}

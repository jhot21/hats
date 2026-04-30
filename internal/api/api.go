package api

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"encoding/json"

	ha "codeberg.org/jhot/go-ha-sdk"
	"codeberg.org/jhot/hats/internal/nats"
	"codeberg.org/jhot/hats/internal/ntfy"
	"codeberg.org/jhot/hats/internal/secrets"
	"codeberg.org/jhot/hats/internal/tokens"
	"codeberg.org/jhot/hats/pkg/config"
	pkgnats "codeberg.org/jhot/hats/pkg/nats"
	ntfyPkg "codeberg.org/jhot/hats/pkg/ntfy"
	"codeberg.org/jhot/passgen"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"codeberg.org/jhot/hats/internal/admin/templates"
)

var (
	cfg        *config.HatsConfig
	logger     *slog.Logger
	server     http.Server
	haClient   *ha.Client
	tokenStore tokens.TokenStore
	secretStore secrets.SecretStore
)

const (
	HA_STATE_PREFIX = "homeassistant.states"
)

func InitAPI(parentLogger *slog.Logger, parentConfig *config.HatsConfig, newRouter func() chi.Router, ts tokens.TokenStore, ss secrets.SecretStore) {
	go func(parentLogger *slog.Logger, parentConfig *config.HatsConfig, newRouter func() chi.Router, ts tokens.TokenStore, ss secrets.SecretStore) {
		for {
			err := Listen(parentLogger, parentConfig, newRouter(), ts, ss)
			if err == nil {
				break
			}
			parentLogger.Error("An error occurred with the API. Restarting now.", "error", err)
			Close()
		}
	}(parentLogger, parentConfig, newRouter, ts, ss)
}

func Listen(parentLogger *slog.Logger, parentConfig *config.HatsConfig, router chi.Router, ts tokens.TokenStore, ss secrets.SecretStore) error {
	logger = parentLogger
	cfg = parentConfig
	tokenStore = ts
	secretStore = ss

	haClient = ha.New(cfg.GetHomeAssistantBaseUrl(), cfg.HomeAssistantToken)

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(loggerMiddleware)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		templates.Home().Render(r.Context(), w)
	})
	router.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "OK")
	})

	router.Get("/passgen", func(w http.ResponseWriter, r *http.Request) {
		passphrase := r.URL.Query().Get("passphrase")
		salt := r.URL.Query().Get("salt")
		lengthString := r.URL.Query().Get("length")
		versionString := r.URL.Query().Get("version")

		length, err := strconv.ParseInt(lengthString, 10, 64)
		if err != nil || length < 1 {
			length = 40
		}

		version := 0 // 0 maps to v3 in upstream (zero value default)
		switch versionString {
		case "1":
			version = 1
		case "2":
			version = 2
		}

		b := passgen.New(passphrase).WithVersion(version).WithSalt(salt).WithLength(int(length))

		if specials := r.URL.Query().Get("specials"); specials != "" {
			b = b.WithSpecials(specials)
		}
		if ambiguous := r.URL.Query().Get("ambiguous"); ambiguous != "" {
			b = b.WithAmbiguousChars(ambiguous)
		} else if r.URL.Query().Has("no-ambiguous") {
			b = b.NoAmbiguous()
		}

		pass, err := b.Generate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		render.PlainText(w, r, pass)
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
		r.Delete("/ntfy", deleteNtfyHandler)
		r.Put("/ntfy/clear", clearNtfyHandler)

		r.Handle("/command/{commandName}", http.HandlerFunc(postCommandHandler))

		r.Get("/secrets", getSecretsHandler)
		r.Get("/secret/{name}", getSecretHandler)
		r.Post("/secret/{name}", createSecretHandler)
		r.Put("/secret/{name}", updateSecretHandler)
		r.Delete("/secret/{name}", deleteSecretHandler)
	})

	server = http.Server{
		Addr:    ":8888",
		Handler: router,
	}

	return server.ListenAndServe()
}

func Close() {
	if server.Addr != "" {
		server.Close()
	}
}

func extractToken(r *http.Request) string {
	_, p, ok := r.BasicAuth()
	if ok && p != "" {
		return p
	}
	parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return ""
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken := extractToken(r)
		if rawToken == "" {
			logger.Warn("Unauthorized request: no token", "method", r.Method, "path", r.URL.Path, "address", r.RemoteAddr)
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}

		var tok *tokens.Token

		if t, err := tokenStore.Validate(rawToken); err == nil {
			tok = t
		} else if cfg.HatsToken != "" && subtle.ConstantTimeCompare([]byte(rawToken), []byte(cfg.HatsToken)) == 1 {
			logger.Warn("HATS_TOKEN is deprecated — create a named token via the admin UI and remove this env var", "method", r.Method, "path", r.URL.Path, "address", r.RemoteAddr)
			tok = &tokens.Token{Name: "HATS_TOKEN (deprecated)", Scopes: []string{"*:*:*"}}
		} else {
			logger.Warn("Unauthorized request: invalid token", "method", r.Method, "path", r.URL.Path, "address", r.RemoteAddr)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		required := tokens.RequiredScope(r)
		if required != "" && !tokens.HasScope(tok.Scopes, required) {
			logger.Warn("Forbidden: insufficient scope", "token", tok.Name, "required", required, "method", r.Method, "path", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			b, _ := json.Marshal(map[string]string{"error": "missing scope: " + required})
			w.Write(b)
			return
		}

		next.ServeHTTP(w, r.WithContext(tokens.ContextWithToken(r.Context(), tok)))
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
			haErr = haClient.CallActionManual(ha.Domain(domain), entityId, ha.Action(service), extras)
		} else {
			l.Debug("Calling service", "extras", extras)
			haErr = haClient.CallAction(entityId, ha.Action(service), extras)
		}
	} else {
		if domain != "" {
			l.Debug("Calling service manually (without extras)")
			haErr = haClient.CallActionManual(ha.Domain(domain), entityId, ha.Action(service))
		} else {
			l.Debug("Calling service (without extras)")
			haErr = haClient.CallAction(entityId, ha.Action(service))
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

	if err := ntfy.Send(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, map[string]string{"id": data.ID})
}

func deleteNtfyHandler(w http.ResponseWriter, r *http.Request) {
	data := &ntfyPkg.Message{}
	if err := render.DecodeJSON(r.Body, data); err != nil {
		http.Error(w, "Unable to parse message data", http.StatusNotAcceptable)
		return
	}
	if err := ntfy.Delete(data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	render.PlainText(w, r, "OK")
}

func clearNtfyHandler(w http.ResponseWriter, r *http.Request) {
	data := &ntfyPkg.Message{}
	if err := render.DecodeJSON(r.Body, data); err != nil {
		http.Error(w, "Unable to parse message data", http.StatusNotAcceptable)
		return
	}
	if err := ntfy.Clear(data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	render.PlainText(w, r, "OK")
}

// Command

// commandPublish is a swappable publish function used for testing.
var commandPublish = func(subject string, data []byte) { nats.Publish(subject, data) }

var strippedHeaders = map[string]struct{}{
	http.CanonicalHeaderKey("Authorization"):       {},
	http.CanonicalHeaderKey("Cookie"):              {},
	http.CanonicalHeaderKey("Proxy-Authorization"): {},
}

func postCommandHandler(w http.ResponseWriter, r *http.Request) {
	commandName := chi.URLParam(r, "commandName")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Error reading request body", "error", err, "url", r.URL.String())
	}

	headers := make(map[string]string, len(r.Header))
	for k, v := range r.Header {
		if _, skip := strippedHeaders[k]; skip {
			continue
		}
		headers[k] = v[0]
	}

	query := make(map[string]string, len(r.URL.Query()))
	for k, v := range r.URL.Query() {
		query[k] = v[0]
	}

	payload := pkgnats.CommandPayload{
		Method:  r.Method,
		Headers: headers,
		Query:   query,
		Body:    body,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Error marshaling command payload", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	commandPublish(fmt.Sprintf("command.%s", commandName), data)
	render.PlainText(w, r, "OK")
}

// SECRETS

func getSecretsHandler(w http.ResponseWriter, r *http.Request) {
	tok := tokens.TokenFromContext(r.Context())
	if tok == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !tokens.HasAnyResourceReadScope(tok.Scopes, "secret") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		b, _ := json.Marshal(map[string]string{"error": "missing scope: read:secret:*"})
		w.Write(b)
		return
	}

	metas, err := secretStore.List()
	if err != nil {
		http.Error(w, "failed to list secrets", http.StatusInternalServerError)
		return
	}

	result := make(map[string]string)
	for _, meta := range metas {
		if !tokens.HasScope(tok.Scopes, "read:secret:"+meta.Name) {
			continue
		}
		s, err := secretStore.Get(meta.Name)
		if err != nil {
			http.Error(w, "failed to retrieve secret "+meta.Name, http.StatusInternalServerError)
			return
		}
		result[meta.Name] = s.Value
	}

	render.JSON(w, r, result)
}

func getSecretHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	s, err := secretStore.Get(name)
	if err != nil {
		if errors.Is(err, secrets.ErrSecretNotFound) {
			http.Error(w, "secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	switch s.Type {
	case secrets.TypeYAML:
		w.Header().Set("Content-Type", "application/x-yaml")
	case secrets.TypeJSON:
		w.Header().Set("Content-Type", "application/json")
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	w.Write([]byte(s.Value))
}

func createSecretHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var body struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := render.DecodeJSON(r.Body, &body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	typ := secrets.SecretType(body.Type)
	switch typ {
	case secrets.TypeBasicString, secrets.TypeEncryptedString, secrets.TypeYAML, secrets.TypeJSON:
	default:
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	if err := secretStore.Create(name, body.Value, typ); err != nil {
		if errors.Is(err, secrets.ErrInvalidName) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, secrets.ErrNameConflict) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.Status(r, http.StatusCreated)
	render.PlainText(w, r, "OK")
}

func updateSecretHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var body struct {
		Value string `json:"value"`
	}
	if err := render.DecodeJSON(r.Body, &body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := secretStore.Update(name, body.Value); err != nil {
		if errors.Is(err, secrets.ErrSecretNotFound) {
			http.Error(w, "secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.PlainText(w, r, "OK")
}

func deleteSecretHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := secretStore.Delete(name); err != nil {
		if errors.Is(err, secrets.ErrSecretNotFound) {
			http.Error(w, "secret not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render.PlainText(w, r, "OK")
}

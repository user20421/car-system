package httpapi

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/car-system/car-system/backend-go/internal/auth"
	"github.com/car-system/car-system/backend-go/internal/domain"
	"github.com/car-system/car-system/backend-go/internal/realtime"
	"github.com/car-system/car-system/backend-go/internal/store"
	"github.com/car-system/car-system/backend-go/internal/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const userKey contextKey = "user"
const sessionKey contextKey = "session"
const deviceKey contextKey = "device"

type Health interface{ Healthy() bool }
type API struct {
	Store     *store.Store
	Auth      *auth.Service
	Redis     *redis.Client
	Queue     Health
	Tickets   *realtime.Tickets
	Hub       *realtime.Hub
	Pipeline  *telemetry.Pipeline
	PublicURL string
	Log       *slog.Logger
	upgrader  websocket.Upgrader
}

func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(a.requestMiddleware)
	r.Get("/health", a.health)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", a.login)
		r.Post("/auth/refresh", a.refresh)
		r.Group(func(r chi.Router) {
			r.Use(a.userMiddleware)
			r.Post("/auth/logout", a.logout)
			r.Get("/auth/me", a.me)
			r.Get("/vehicles", a.vehicles)
			r.Get("/vehicles/{vin}", a.vehicle)
			r.Get("/vehicles/{vin}/telemetry/latest", a.latest)
			r.Get("/vehicles/{vin}/telemetry/history", a.history)
			r.Post("/vehicles/{vin}/commands", a.createCommand)
			r.Get("/vehicles/{vin}/commands/{commandID}", a.command)
			r.Get("/alerts", a.alerts)
			r.Post("/alerts/{alertID}/ack", a.ackAlert)
			r.Post("/ws-ticket", a.wsTicket)
		})
		r.Group(func(r chi.Router) {
			r.Use(a.deviceMiddleware)
			r.Get("/device/config", a.deviceConfig)
			r.Get("/device/ota/latest", a.otaLatest)
			r.Post("/device/telemetry/batch", a.telemetryBatch)
		})
		r.Get("/ws", a.websocket)
	})
	return r
}
func (a *API) requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if len(id) < 8 || len(id) > 128 {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		ctx := context.WithValue(r.Context(), contextKey("request"), id)
		start := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				a.Log.Error("panic", "request_id", id, "error", recovered)
				problem(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", "")
			} else {
				a.Log.Debug("request", "request_id", id, "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
			}
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func requestID(ctx context.Context) string {
	id, _ := ctx.Value(contextKey("request")).(string)
	return id
}
func problem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://api.car.local/problems/" + strings.ToLower(code), "title": title, "status": status, "detail": detail, "code": code, "request_id": requestID(r.Context())})
}
func fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		problem(w, r, 400, "VALIDATION_ERROR", "Request validation failed", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		problem(w, r, 403, "FORBIDDEN", "Forbidden", "")
	case errors.Is(err, domain.ErrNotFound):
		problem(w, r, 404, "NOT_FOUND", "Resource not found", "")
	case errors.Is(err, domain.ErrConflict):
		problem(w, r, 409, "CONFLICT", "Resource conflict", err.Error())
	default:
		problem(w, r, 500, "INTERNAL_ERROR", "Internal server error", "")
	}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any, max int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, max)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		problem(w, r, 400, "VALIDATION_ERROR", "Request validation failed", err.Error())
		return false
	}
	return true
}
func user(ctx context.Context) domain.User { u, _ := ctx.Value(userKey).(domain.User); return u }
func (a *API) userMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if len(header) < 8 || !strings.EqualFold(header[:7], "Bearer ") {
			problem(w, r, 401, "UNAUTHORIZED", "Unauthorized", "")
			return
		}
		u, sid, err := a.Auth.Authenticate(r.Context(), strings.TrimSpace(header[7:]))
		if err != nil {
			problem(w, r, 401, "UNAUTHORIZED", "Unauthorized", "")
			return
		}
		ctx := context.WithValue(r.Context(), userKey, u)
		ctx = context.WithValue(ctx, sessionKey, sid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func (a *API) deviceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
			problem(w, r, 401, "DEVICE_CERTIFICATE_REQUIRED", "Device certificate required", "")
			return
		}
		cert := r.TLS.PeerCertificates[0]
		if !validClientCertificate(cert, r.TLS.VerifiedChains) {
			problem(w, r, 401, "INVALID_DEVICE_CERTIFICATE", "Invalid device certificate", "")
			return
		}
		vin := cert.Subject.CommonName
		if !domain.ValidVIN(vin) {
			problem(w, r, 403, "INVALID_DEVICE_IDENTITY", "Invalid device identity", "")
			return
		}
		if _, err := a.Store.VehicleByVIN(r.Context(), vin); err != nil {
			problem(w, r, 403, "UNKNOWN_DEVICE", "Unknown device", "")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), deviceKey, vin)))
	})
}
func validClientCertificate(cert *x509.Certificate, chains [][]*x509.Certificate) bool {
	return cert != nil && len(chains) > 0
}

const refreshCookieName = "aurora_refresh"

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookieName, Value: token, Path: "/api/v1/auth", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: 30 * 24 * 60 * 60})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookieName, Value: "", Path: "/api/v1/auth", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req, 16384) {
		return
	}
	tokens, err := a.Auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		problem(w, r, 401, "INVALID_CREDENTIALS", "Invalid credentials", "")
		return
	}
	setRefreshCookie(w, tokens.RefreshToken)
	writeJSON(w, 200, tokens)
}
func (a *API) refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if r.ContentLength > 0 && !decodeJSON(w, r, &req, 16384) {
		return
	}
	if req.RefreshToken == "" {
		cookie, err := r.Cookie(refreshCookieName)
		if err != nil || cookie.Value == "" {
			problem(w, r, 400, "VALIDATION_ERROR", "Request validation failed", "refresh_token or refresh cookie required")
			return
		}
		// A custom header forces a CORS preflight for cross-site requests. The
		// SameSite=Strict cookie is therefore never sufficient by itself for CSRF.
		if r.Header.Get("X-Request-ID") == "" {
			problem(w, r, http.StatusForbidden, "CSRF_REJECTED", "Request origin validation failed", "")
			return
		}
		req.RefreshToken = cookie.Value
	}
	tokens, err := a.Auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		problem(w, r, 401, "INVALID_REFRESH_TOKEN", "Invalid refresh token", "")
		return
	}
	setRefreshCookie(w, tokens.RefreshToken)
	writeJSON(w, 200, tokens)
}
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	sid, _ := r.Context().Value(sessionKey).(string)
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if r.ContentLength > 0 {
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384)).Decode(&req)
	}
	if err := a.Auth.Logout(r.Context(), sid); err != nil {
		fail(w, r, err)
		return
	}
	clearRefreshCookie(w)
	w.WriteHeader(204)
}
func (a *API) me(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, user(r.Context())) }
func pageParams(r *http.Request) (int, int, error) {
	page, size := 1, 20
	var err error
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 {
			return 0, 0, domain.ErrInvalid
		}
	}
	if raw := r.URL.Query().Get("page_size"); raw != "" {
		size, err = strconv.Atoi(raw)
		if err != nil || size < 1 || size > 100 {
			return 0, 0, domain.ErrInvalid
		}
	}
	return page, size, nil
}
func pageMeta(page, size, total int) map[string]int {
	pages := 0
	if total > 0 {
		pages = (total + size - 1) / size
	}
	return map[string]int{"number": page, "size": size, "total_items": total, "total_pages": pages}
}
func (a *API) vehicles(w http.ResponseWriter, r *http.Request) {
	page, size, err := pageParams(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	items, total, err := a.Store.ListVehicles(r.Context(), user(r.Context()), page, size)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "page": pageMeta(page, size, total)})
}
func (a *API) vehicle(w http.ResponseWriter, r *http.Request) {
	v, err := a.Store.VehicleForUser(r.Context(), user(r.Context()), chi.URLParam(r, "vin"))
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, 200, v)
}
func (a *API) latest(w http.ResponseWriter, r *http.Request) {
	vin := chi.URLParam(r, "vin")
	if _, err := a.Store.VehicleForUser(r.Context(), user(r.Context()), vin); err != nil {
		fail(w, r, err)
		return
	}
	t, err := telemetry.Latest(r.Context(), a.Redis, vin)
	if errors.Is(err, redis.Nil) {
		t, err = a.Store.LatestTelemetry(r.Context(), vin)
	}
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, 200, t)
}
func (a *API) history(w http.ResponseWriter, r *http.Request) {
	vin := chi.URLParam(r, "vin")
	if _, err := a.Store.VehicleForUser(r.Context(), user(r.Context()), vin); err != nil {
		fail(w, r, err)
		return
	}
	from, err1 := strconv.ParseInt(r.URL.Query().Get("from_ms"), 10, 64)
	to, err2 := strconv.ParseInt(r.URL.Query().Get("to_ms"), 10, 64)
	if err1 != nil || err2 != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	if err := store.ValidateTimeRange(from, to); err != nil {
		fail(w, r, err)
		return
	}
	limit, err := store.ParseLimit(r.URL.Query().Get("limit"), 200, 1000)
	if err != nil {
		fail(w, r, err)
		return
	}
	items, next, err := a.Store.TelemetryHistory(r.Context(), vin, from, to, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "next_cursor": next})
}
func (a *API) createCommand(w http.ResponseWriter, r *http.Request) {
	vin := chi.URLParam(r, "vin")
	var req struct {
		IdempotencyKey string            `json:"idempotency_key"`
		ExpiresAtMS    int64             `json:"expires_at_ms"`
		Type           string            `json:"type"`
		Params         map[string]string `json:"params"`
	}
	if !decodeJSON(w, r, &req, 65536) {
		return
	}
	if err := domain.ValidateCommandRequest(req.IdempotencyKey, req.Type, req.Params, req.ExpiresAtMS, time.Now().UnixMilli()); err != nil {
		fail(w, r, err)
		return
	}
	c, err := a.Store.CreateCommand(r.Context(), user(r.Context()), vin, req.IdempotencyKey, req.Type, req.Params, req.ExpiresAtMS)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, 202, c)
}
func (a *API) command(w http.ResponseWriter, r *http.Request) {
	c, err := a.Store.CommandForUser(r.Context(), user(r.Context()), chi.URLParam(r, "vin"), chi.URLParam(r, "commandID"))
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, 200, c)
}
func (a *API) alerts(w http.ResponseWriter, r *http.Request) {
	page, size, err := pageParams(r)
	if err != nil {
		fail(w, r, err)
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != "open" && status != "acknowledged" {
		fail(w, r, domain.ErrInvalid)
		return
	}
	items, total, err := a.Store.ListAlerts(r.Context(), user(r.Context()), r.URL.Query().Get("vin"), status, page, size)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "page": pageMeta(page, size, total)})
}
func (a *API) ackAlert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Note string `json:"note"`
	}
	if r.ContentLength > 0 && !decodeJSON(w, r, &req, 16384) {
		return
	}
	if len(req.Note) > 1000 {
		fail(w, r, domain.ErrInvalid)
		return
	}
	alert, err := a.Store.AcknowledgeAlert(r.Context(), user(r.Context()), chi.URLParam(r, "alertID"), req.Note, time.Now().UnixMilli())
	if err != nil {
		fail(w, r, err)
		return
	}
	a.Hub.Alert(alert)
	writeJSON(w, 200, alert)
}
func (a *API) wsTicket(w http.ResponseWriter, r *http.Request) {
	token, expires, err := a.Tickets.Issue(user(r.Context()).ID)
	if err != nil {
		fail(w, r, err)
		return
	}
	parsed, _ := url.Parse(a.PublicURL)
	parsed.Scheme = "wss"
	parsed.Path = "/api/v1/ws"
	writeJSON(w, 201, map[string]any{"ticket": token, "expires_at_ms": expires, "websocket_url": parsed.String()})
}

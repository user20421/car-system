package httpapi

import (
	"context"
	"net/http"
	"time"
)

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	checks := map[string]string{"mysql": "up", "redis": "up", "rabbitmq": "up"}
	if err := a.Store.Ping(ctx); err != nil {
		checks["mysql"] = "down"
	}
	if err := a.Redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "down"
	}
	if a.Queue == nil || !a.Queue.Healthy() {
		checks["rabbitmq"] = "down"
	}
	for _, status := range checks {
		if status == "down" {
			problem(w, r, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "Dependency unavailable", "")
			return
		}
	}
	writeJSON(w, 200, map[string]any{"status": "ok", "version": "1.0.0", "checks": checks})
}

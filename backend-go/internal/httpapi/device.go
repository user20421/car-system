package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
)

func (a *API) deviceConfig(w http.ResponseWriter, r *http.Request) {
	vin := r.Context().Value(deviceKey).(string)
	cfg, err := a.Store.DeviceConfig(r.Context(), vin)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, 200, cfg)
}
func (a *API) otaLatest(w http.ResponseWriter, r *http.Request) {
	vin := r.Context().Value(deviceKey).(string)
	current := r.URL.Query().Get("current_version")
	if !validVersion(current) {
		fail(w, r, fmt.Errorf("%w: invalid current_version", domain.ErrInvalid))
		return
	}
	release, err := a.Store.LatestOTA(r.Context(), vin)
	if err != nil {
		fail(w, r, err)
		return
	}
	available := release != nil && compareVersion(release.Version, current) > 0
	writeJSON(w, 200, map[string]any{"update_available": available, "release": func() any {
		if available {
			return release
		}
		return nil
	}()})
}
func validVersion(v string) bool {
	if v == "" || len(v) > 64 {
		return false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.ParseUint(part, 10, 32); err != nil {
			return false
		}
	}
	return true
}
func compareVersion(a, b string) int {
	aa, bb := strings.Split(a, "."), strings.Split(b, ".")
	if len(aa) != 3 {
		return -1
	}
	for i := 0; i < 3; i++ {
		x, _ := strconv.Atoi(aa[i])
		y, _ := strconv.Atoi(bb[i])
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

type batchRequest struct {
	BatchID      string             `json:"batch_id"`
	VIN          string             `json:"vin"`
	DeviceBootID string             `json:"device_boot_id"`
	Items        []domain.Telemetry `json:"items"`
}

func (a *API) telemetryBatch(w http.ResponseWriter, r *http.Request) {
	vin := r.Context().Value(deviceKey).(string)
	var req batchRequest
	if !decodeJSON(w, r, &req, 4<<20) {
		return
	}
	if len(req.BatchID) < 8 || len(req.BatchID) > 128 || req.VIN != vin || len(req.Items) < 1 || len(req.Items) > 500 {
		fail(w, r, fmt.Errorf("%w: invalid telemetry batch", domain.ErrInvalid))
		return
	}
	now := time.Now().UTC()
	for i := range req.Items {
		item := &req.Items[i]
		if item.VIN != vin || item.DeviceBootID != req.DeviceBootID {
			fail(w, r, fmt.Errorf("%w: batch item identity mismatch", domain.ErrInvalid))
			return
		}
		if item.ReceivedAtMS == 0 {
			item.ReceivedAtMS = now.UnixMilli()
		}
		if err := domain.ValidateTelemetry(*item); err != nil {
			fail(w, r, err)
			return
		}
	}
	accepted, duplicates, err := a.Store.InsertOfflineBatch(r.Context(), vin, req.BatchID, req.Items, now.UnixMilli())
	if err != nil {
		fail(w, r, err)
		return
	}
	a.Pipeline.PublishPersisted(req.Items)
	writeJSON(w, 202, map[string]any{"batch_id": req.BatchID, "accepted": accepted, "duplicates": duplicates})
}

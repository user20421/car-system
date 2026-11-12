package domain

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalid      = errors.New("invalid")
)

var vinPattern = regexp.MustCompile(`^[A-Z0-9]{17}$`)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	DisplayName  string    `json:"display_name"`
	Roles        []string  `json:"roles"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func (u User) Admin() bool {
	for _, role := range u.Roles {
		if role == "admin" {
			return true
		}
	}
	return false
}

type Vehicle struct {
	VIN             string    `json:"vin"`
	DisplayName     string    `json:"display_name"`
	Model           string    `json:"model"`
	ModelYear       int       `json:"model_year"`
	OwnerUserID     string    `json:"owner_user_id"`
	AccessRole      string    `json:"access_role"`
	FirmwareVersion string    `json:"firmware_version,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type Location struct {
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
	AltitudeM           float64 `json:"altitude_m,omitempty"`
	HeadingDeg          float64 `json:"heading_deg,omitempty"`
	HorizontalAccuracyM float64 `json:"horizontal_accuracy_m,omitempty"`
}

type HVAC struct {
	Mode          string  `json:"mode,omitempty"`
	TargetTempC   float64 `json:"target_temp_c,omitempty"`
	FanLevel      uint32  `json:"fan_level,omitempty"`
	ACOn          bool    `json:"ac_on,omitempty"`
	Recirculation bool    `json:"recirculation,omitempty"`
}

type Fault struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message,omitempty"`
	Active   bool   `json:"active"`
}

type Telemetry struct {
	MessageID    string    `json:"message_id,omitempty"`
	VIN          string    `json:"vin"`
	DeviceBootID string    `json:"device_boot_id"`
	Sequence     uint64    `json:"sequence"`
	OccurredAtMS int64     `json:"occurred_at_ms"`
	ReceivedAtMS int64     `json:"received_at_ms,omitempty"`
	SpeedKPH     float64   `json:"speed_kph"`
	RPM          uint32    `json:"rpm"`
	Battery      float64   `json:"battery_percent"`
	RangeKM      float64   `json:"range_km"`
	Location     *Location `json:"location,omitempty"`
	Locked       bool      `json:"locked"`
	HVAC         *HVAC     `json:"hvac,omitempty"`
	CabinTempC   float64   `json:"cabin_temp_c,omitempty"`
	Defrost      bool      `json:"defrost"`
	Fault        []Fault   `json:"fault"`
}

type CommandError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Retryable bool              `json:"retryable"`
	Details   map[string]string `json:"details,omitempty"`
}

type Command struct {
	CommandID      string            `json:"command_id"`
	VIN            string            `json:"vin"`
	RequestedBy    string            `json:"requested_by"`
	IdempotencyKey string            `json:"idempotency_key"`
	ExpiresAtMS    int64             `json:"expires_at_ms"`
	Type           string            `json:"type"`
	Params         map[string]string `json:"params"`
	Status         string            `json:"status"`
	Error          *CommandError     `json:"error,omitempty"`
	CreatedAtMS    int64             `json:"created_at_ms"`
	UpdatedAtMS    int64             `json:"updated_at_ms"`
}

type Alert struct {
	ID                  string  `json:"id"`
	VIN                 string  `json:"vin"`
	Code                string  `json:"code"`
	Title               string  `json:"title"`
	Severity            string  `json:"severity"`
	Status              string  `json:"status"`
	OccurredAtMS        int64   `json:"occurred_at_ms"`
	AcknowledgedBy      *string `json:"acknowledged_by"`
	AcknowledgedAtMS    *int64  `json:"acknowledged_at_ms"`
	AcknowledgementNote *string `json:"acknowledgement_note"`
}

func ValidVIN(vin string) bool { return vinPattern.MatchString(vin) }

func CanView(user User, vehicle Vehicle, share string) bool {
	return user.Admin() || vehicle.OwnerUserID == user.ID || share == "driver" || share == "viewer"
}

func CanControl(user User, vehicle Vehicle, share string) bool {
	return user.Admin() || vehicle.OwnerUserID == user.ID || share == "driver"
}

func CanAcknowledge(user User, vehicle Vehicle, share string) bool {
	return CanControl(user, vehicle, share)
}

func ValidateTelemetry(t Telemetry) error {
	if !ValidVIN(t.VIN) || t.DeviceBootID == "" || len(t.DeviceBootID) > 128 || t.MessageID == "" || len(t.MessageID) > 128 {
		return fmt.Errorf("%w: invalid envelope", ErrInvalid)
	}
	if t.OccurredAtMS <= 0 {
		return fmt.Errorf("%w: occurred_at_ms must be positive", ErrInvalid)
	}
	values := []struct {
		name            string
		value, min, max float64
	}{
		{"speed_kph", t.SpeedKPH, 0, 500},
		{"battery_percent", t.Battery, 0, 100},
		{"range_km", t.RangeKM, 0, 10000},
		{"cabin_temp_c", t.CabinTempC, -100, 150},
	}
	for _, v := range values {
		if math.IsNaN(v.value) || math.IsInf(v.value, 0) || v.value < v.min || v.value > v.max {
			return fmt.Errorf("%w: %s outside [%g,%g]", ErrInvalid, v.name, v.min, v.max)
		}
	}
	if t.Location != nil {
		l := t.Location
		if invalidFloat(l.Latitude, -90, 90) || invalidFloat(l.Longitude, -180, 180) || invalidFloat(l.HeadingDeg, 0, 360) || invalidFloat(l.HorizontalAccuracyM, 0, 100000) || math.IsNaN(l.AltitudeM) || math.IsInf(l.AltitudeM, 0) {
			return fmt.Errorf("%w: invalid location", ErrInvalid)
		}
	}
	if len(t.Fault) > 100 {
		return fmt.Errorf("%w: too many faults", ErrInvalid)
	}
	for _, f := range t.Fault {
		if f.Code == "" || len(f.Code) > 64 || (f.Severity != "info" && f.Severity != "warning" && f.Severity != "critical") {
			return fmt.Errorf("%w: invalid fault", ErrInvalid)
		}
	}
	return nil
}

func invalidFloat(v, min, max float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 0) || v < min || v > max
}

var commandTypes = map[string]bool{"lock": true, "unlock": true, "hvac_on": true, "hvac_off": true, "set_temperature": true, "defrost_on": true, "defrost_off": true, "honk_flash": true}

func ValidateCommandRequest(key, kind string, params map[string]string, expiresAt, now int64) error {
	if len(key) < 8 || len(key) > 128 || !commandTypes[kind] {
		return fmt.Errorf("%w: invalid idempotency key or command type", ErrInvalid)
	}
	if expiresAt <= now || expiresAt > now+int64((24*time.Hour)/time.Millisecond) {
		return fmt.Errorf("%w: expiry must be in the future and within 24h", ErrInvalid)
	}
	if len(params) > 32 {
		return fmt.Errorf("%w: too many parameters", ErrInvalid)
	}
	for k, v := range params {
		if k == "" || len(k) > 64 || len(v) > 256 {
			return fmt.Errorf("%w: invalid parameter", ErrInvalid)
		}
	}
	if kind == "set_temperature" {
		var temp float64
		if _, err := fmt.Sscan(params["target_temp_c"], &temp); err != nil || invalidFloat(temp, 16, 30) {
			return fmt.Errorf("%w: target_temp_c must be 16..30", ErrInvalid)
		}
	}
	return nil
}

func CanonicalCommand(kind string, params map[string]string, expiresAt int64) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(kind)
	fmt.Fprintf(&b, "\n%d", expiresAt)
	for _, k := range keys {
		fmt.Fprintf(&b, "\n%s=%s", k, params[k])
	}
	return b.String()
}

var transitions = map[string]map[string]bool{
	"pending":   {"sent": true, "expired": true, "failed": true},
	"sent":      {"received": true, "expired": true, "failed": true},
	"received":  {"succeeded": true, "failed": true, "expired": true},
	"succeeded": {}, "failed": {}, "expired": {},
}

func CanTransition(from, to string) bool { return transitions[from][to] }

func Transition(current, next string) (string, error) {
	if current == next || CanTransition(current, next) {
		return next, nil
	}
	return current, fmt.Errorf("%w: illegal command transition %s -> %s", ErrConflict, current, next)
}

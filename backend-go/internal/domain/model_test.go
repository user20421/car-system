package domain

import (
	"math"
	"testing"
	"time"
)

func TestRBACAndABAC(t *testing.T) {
	admin := User{ID: "a", Roles: []string{"admin"}}
	owner := User{ID: "o", Roles: []string{"user"}}
	other := User{ID: "x", Roles: []string{"user"}}
	vehicle := Vehicle{OwnerUserID: "o"}
	if !CanView(admin, vehicle, "") || !CanControl(admin, vehicle, "") {
		t.Fatal("admin must manage all vehicles")
	}
	if !CanView(owner, vehicle, "") || !CanControl(owner, vehicle, "") {
		t.Fatal("owner access denied")
	}
	if !CanView(other, vehicle, "viewer") || CanControl(other, vehicle, "viewer") {
		t.Fatal("viewer permissions incorrect")
	}
	if !CanView(other, vehicle, "driver") || !CanControl(other, vehicle, "driver") {
		t.Fatal("driver permissions incorrect")
	}
	if CanView(other, vehicle, "") || CanControl(other, vehicle, "") {
		t.Fatal("unshared vehicle exposed")
	}
}
func TestCommandStateMachine(t *testing.T) {
	for _, test := range []struct {
		from, to string
		ok       bool
	}{{"pending", "sent", true}, {"sent", "received", true}, {"received", "succeeded", true}, {"pending", "succeeded", false}, {"succeeded", "failed", false}, {"sent", "expired", true}, {"received", "received", true}} {
		got, err := Transition(test.from, test.to)
		if (err == nil) != test.ok {
			t.Errorf("%s -> %s: err=%v", test.from, test.to, err)
		}
		if test.ok && got != test.to {
			t.Errorf("got %s", got)
		}
	}
}
func TestCommandFingerprintCanonical(t *testing.T) {
	a := CanonicalCommand("lock", map[string]string{"b": "2", "a": "1"}, 123)
	b := CanonicalCommand("lock", map[string]string{"a": "1", "b": "2"}, 123)
	if a != b {
		t.Fatal("map order changed request fingerprint")
	}
	if a == CanonicalCommand("lock", map[string]string{"a": "1", "b": "3"}, 123) {
		t.Fatal("different request has same fingerprint")
	}
}
func validTelemetry() Telemetry {
	return Telemetry{MessageID: "m1", VIN: "LTEST000000000001", DeviceBootID: "boot", OccurredAtMS: time.Now().UnixMilli(), Battery: 50, Fault: []Fault{}}
}
func TestTelemetryValidation(t *testing.T) {
	if err := ValidateTelemetry(validTelemetry()); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Telemetry){func(v *Telemetry) { v.VIN = "bad" }, func(v *Telemetry) { v.Battery = 101 }, func(v *Telemetry) { v.SpeedKPH = math.NaN() }, func(v *Telemetry) { v.OccurredAtMS = 0 }, func(v *Telemetry) { v.Location = &Location{Latitude: 91, Longitude: 0} }, func(v *Telemetry) { v.Fault = []Fault{{Code: "x", Severity: "fatal"}} }} {
		v := validTelemetry()
		mutate(&v)
		if err := ValidateTelemetry(v); err == nil {
			t.Fatal("invalid telemetry accepted")
		}
	}
}

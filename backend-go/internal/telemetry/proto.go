package telemetry

import (
	"fmt"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
	vehiclev1 "github.com/car-system/car-system/backend-go/pkg/vehiclev1"
	"google.golang.org/protobuf/proto"
)

func Decode(payload []byte, expectedVIN string, now time.Time) (domain.Telemetry, error) {
	var message vehiclev1.Telemetry
	if err := proto.Unmarshal(payload, &message); err != nil {
		return domain.Telemetry{}, fmt.Errorf("%w: protobuf: %v", domain.ErrInvalid, err)
	}
	return FromProto(&message, expectedVIN, now)
}
func FromProto(message *vehiclev1.Telemetry, expectedVIN string, now time.Time) (domain.Telemetry, error) {
	if message == nil || message.Envelope == nil {
		return domain.Telemetry{}, fmt.Errorf("%w: envelope required", domain.ErrInvalid)
	}
	e := message.Envelope
	if e.Version != "v1" || e.Vin != expectedVIN || e.MessageId == "" {
		return domain.Telemetry{}, fmt.Errorf("%w: invalid envelope identity", domain.ErrInvalid)
	}
	if e.OccurredAtMs > now.Add(5*time.Minute).UnixMilli() || e.OccurredAtMs < now.Add(-90*24*time.Hour).UnixMilli() {
		return domain.Telemetry{}, fmt.Errorf("%w: telemetry timestamp outside acceptance window", domain.ErrInvalid)
	}
	t := domain.Telemetry{MessageID: e.MessageId, VIN: e.Vin, DeviceBootID: e.DeviceBootId, Sequence: e.Sequence, OccurredAtMS: e.OccurredAtMs, ReceivedAtMS: now.UnixMilli(), SpeedKPH: message.SpeedKph, RPM: message.Rpm, Battery: message.BatteryPercent, RangeKM: message.RangeKm, Locked: message.Locked, CabinTempC: message.CabinTempC, Defrost: message.Defrost, Fault: []domain.Fault{}}
	if message.Location != nil {
		l := message.Location
		t.Location = &domain.Location{Latitude: l.Latitude, Longitude: l.Longitude, AltitudeM: l.AltitudeM, HeadingDeg: l.HeadingDeg, HorizontalAccuracyM: l.HorizontalAccuracyM}
	}
	if message.Hvac != nil {
		h := message.Hvac
		t.HVAC = &domain.HVAC{Mode: hvacMode(h.Mode), TargetTempC: h.TargetTempC, FanLevel: h.FanLevel, ACOn: h.AcOn, Recirculation: h.Recirculation}
	}
	for _, f := range message.Fault {
		t.Fault = append(t.Fault, domain.Fault{Code: f.Code, Severity: severity(f.Severity), Message: f.Message, Active: f.Active})
	}
	return t, domain.ValidateTelemetry(t)
}
func hvacMode(mode vehiclev1.HvacState_Mode) string {
	switch mode {
	case vehiclev1.HvacState_MODE_OFF:
		return "off"
	case vehiclev1.HvacState_MODE_AUTO:
		return "auto"
	case vehiclev1.HvacState_MODE_COOL:
		return "cool"
	case vehiclev1.HvacState_MODE_HEAT:
		return "heat"
	default:
		return ""
	}
}
func severity(value vehiclev1.Fault_Severity) string {
	switch value {
	case vehiclev1.Fault_SEVERITY_INFO:
		return "info"
	case vehiclev1.Fault_SEVERITY_WARNING:
		return "warning"
	case vehiclev1.Fault_SEVERITY_CRITICAL:
		return "critical"
	default:
		return ""
	}
}

package telemetry

import (
	"testing"
	"time"

	vehiclev1 "github.com/car-system/car-system/backend-go/pkg/vehiclev1"
	"google.golang.org/protobuf/proto"
)

func TestProtobufTelemetryRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	input := &vehiclev1.Telemetry{Envelope: &vehiclev1.Envelope{Version: "v1", MessageId: "message-1", Vin: "LTEST000000000001", DeviceBootId: "boot-1", Sequence: 9, OccurredAtMs: now.UnixMilli()}, SpeedKph: 88.5, BatteryPercent: 76, Location: &vehiclev1.Location{Latitude: 31.2, Longitude: 121.5}, Hvac: &vehiclev1.HvacState{Mode: vehiclev1.HvacState_MODE_AUTO}, Fault: []*vehiclev1.Fault{{Code: "BMS_1", Severity: vehiclev1.Fault_SEVERITY_WARNING, Active: true}}}
	raw, err := proto.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(raw, input.Envelope.Vin, now)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.MessageID != "message-1" || decoded.Sequence != 9 || decoded.SpeedKPH != 88.5 || decoded.HVAC.Mode != "auto" || decoded.Fault[0].Severity != "warning" {
		t.Fatalf("round trip mismatch: %+v", decoded)
	}
	var second vehiclev1.Telemetry
	if err := proto.Unmarshal(raw, &second); err != nil || !proto.Equal(input, &second) {
		t.Fatal("generated protobuf is not stable")
	}
}
func TestProtobufRejectsVINAndFutureTime(t *testing.T) {
	now := time.Now()
	message := &vehiclev1.Telemetry{Envelope: &vehiclev1.Envelope{Version: "v1", MessageId: "m", Vin: "LTEST000000000001", DeviceBootId: "b", OccurredAtMs: now.Add(time.Hour).UnixMilli()}}
	if _, err := FromProto(message, message.Envelope.Vin, now); err == nil {
		t.Fatal("future telemetry accepted")
	}
	message.Envelope.OccurredAtMs = now.UnixMilli()
	if _, err := FromProto(message, "LTEST000000000002", now); err == nil {
		t.Fatal("certificate/topic VIN mismatch accepted")
	}
}

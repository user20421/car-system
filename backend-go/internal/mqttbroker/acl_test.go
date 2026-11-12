package mqttbroker

import "testing"

func TestVehicleTopicACL(t *testing.T) {
	vin := "LTEST000000000001"
	for _, topic := range []string{"vehicles/" + vin + "/telemetry/v1", "vehicles/" + vin + "/ack/v1", "vehicles/" + vin + "/status/v1"} {
		if !allowedTopic(vin, topic, true) {
			t.Errorf("device cannot publish %s", topic)
		}
	}
	if allowedTopic(vin, "vehicles/"+vin+"/command/v1", true) {
		t.Fatal("device may not publish commands")
	}
	if !allowedTopic(vin, "vehicles/"+vin+"/command/v1", false) {
		t.Fatal("device cannot subscribe command")
	}
	if allowedTopic(vin, "vehicles/LTEST000000000002/telemetry/v1", true) {
		t.Fatal("cross-VIN publish allowed")
	}
	if allowedTopic(vin, "vehicles/"+vin+"/#", false) {
		t.Fatal("wildcard ACL allowed")
	}
}

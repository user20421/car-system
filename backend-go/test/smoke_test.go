package integration_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
	vehiclev1 "github.com/car-system/car-system/backend-go/pkg/vehiclev1"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}
type smokeClient struct {
	t      *testing.T
	client *http.Client
	base   string
}

func TestLiveSmoke(t *testing.T) {
	if os.Getenv("CAR_INTEGRATION") != "1" {
		t.Skip("set CAR_INTEGRATION=1 with the TLS dependencies and server running")
	}
	ca, err := os.ReadFile("../../deploy/certs/ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		t.Fatal("invalid CA")
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots}
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig, ForceAttemptHTTP2: true}, Timeout: 10 * time.Second}
	s := &smokeClient{t: t, client: client, base: "https://localhost:8443"}
	response, err := client.Get(s.base + "/health")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 || response.ProtoMajor != 2 {
		t.Fatalf("health status=%d protocol=%s", response.StatusCode, response.Proto)
	}
	var tokens tokenResponse
	s.json("POST", "/api/v1/auth/login", "", map[string]string{"username": "owner", "password": "Owner123!"}, 200, &tokens)
	if tokens.ExpiresIn != 900 {
		t.Fatal("wrong access TTL")
	}
	var vehicles struct {
		Items []domain.Vehicle `json:"items"`
	}
	s.json("GET", "/api/v1/vehicles", tokens.AccessToken, nil, 200, &vehicles)
	if len(vehicles.Items) != 2 {
		t.Fatalf("owner sees %d vehicles", len(vehicles.Items))
	}
	key := "smoke-" + uuid.NewString()
	request := map[string]any{"idempotency_key": key, "expires_at_ms": time.Now().Add(time.Minute).UnixMilli(), "type": "lock", "params": map[string]string{}}
	var command, repeat domain.Command
	s.json("POST", "/api/v1/vehicles/LTEST000000000001/commands", tokens.AccessToken, request, 202, &command)
	s.json("POST", "/api/v1/vehicles/LTEST000000000001/commands", tokens.AccessToken, request, 202, &repeat)
	if command.CommandID != repeat.CommandID {
		t.Fatal("idempotency returned another command")
	}
	s.json("GET", "/api/v1/vehicles/LTEST000000000001/commands/"+command.CommandID, tokens.AccessToken, nil, 200, &command)
	var ticket struct {
		Ticket string `json:"ticket"`
		URL    string `json:"websocket_url"`
	}
	s.json("POST", "/api/v1/ws-ticket", tokens.AccessToken, nil, 201, &ticket)
	wsTLS := tlsConfig.Clone()
	wsTLS.NextProtos = []string{"http/1.1"}
	dialer := websocket.Dialer{TLSClientConfig: wsTLS, HandshakeTimeout: 5 * time.Second}
	ws, _, err := dialer.DialContext(context.Background(), ticket.URL+"?ticket="+ticket.Ticket, nil)
	if err != nil {
		t.Fatal(err)
	}
	ws.Close()
	if _, response, err := dialer.Dial(ticket.URL+"?ticket="+ticket.Ticket, nil); err == nil {
		t.Fatal("WSS ticket was reusable")
	} else if response != nil {
		response.Body.Close()
	}
	var rotated tokenResponse
	s.json("POST", "/api/v1/auth/refresh", "", map[string]string{"refresh_token": tokens.RefreshToken}, 200, &rotated)
	if rotated.RefreshToken == tokens.RefreshToken {
		t.Fatal("refresh token did not rotate")
	}
	var problem map[string]any
	s.json("POST", "/api/v1/auth/refresh", "", map[string]string{"refresh_token": tokens.RefreshToken}, 401, &problem)
	deviceCert, err := tls.LoadX509KeyPair("../../deploy/certs/LTEST000000000001.crt", "../../deploy/certs/LTEST000000000001.key")
	if err != nil {
		t.Fatal(err)
	}
	deviceHTTP := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, Certificates: []tls.Certificate{deviceCert}}}, Timeout: 10 * time.Second}
	device := &smokeClient{t: t, client: deviceHTTP, base: s.base}
	var config map[string]any
	device.json("GET", "/api/v1/device/config", "", nil, 200, &config)
	if config["vin"] != "LTEST000000000001" {
		t.Fatal("wrong device identity")
	}
	messageID := uuid.NewString()
	item := domain.Telemetry{MessageID: messageID, VIN: "LTEST000000000001", DeviceBootID: "smoke-" + uuid.NewString(), Sequence: 1, OccurredAtMS: time.Now().UnixMilli(), Battery: 80, Fault: []domain.Fault{}}
	batch := map[string]any{"batch_id": "smoke-" + uuid.NewString(), "vin": item.VIN, "device_boot_id": item.DeviceBootID, "items": []domain.Telemetry{item}}
	var accepted struct {
		Accepted   int `json:"accepted"`
		Duplicates int `json:"duplicates"`
	}
	device.json("POST", "/api/v1/device/telemetry/batch", "", batch, 202, &accepted)
	if accepted.Accepted != 1 {
		t.Fatalf("batch result: %#v", accepted)
	}
	var active tokenResponse
	s.json("POST", "/api/v1/auth/login", "", map[string]string{"username": "owner", "password": "Owner123!"}, 200, &active)
	var latest domain.Telemetry
	s.json("GET", "/api/v1/vehicles/LTEST000000000001/telemetry/latest", active.AccessToken, nil, 200, &latest)
	if latest.MessageID != messageID {
		t.Fatal("latest telemetry was not updated")
	}

	// A real vehicle can ACK only after the dispatcher has delivered the command.
	dispatchDeadline := time.Now().Add(5 * time.Second)
	for command.Status == "pending" && time.Now().Before(dispatchDeadline) {
		time.Sleep(50 * time.Millisecond)
		s.json("GET", "/api/v1/vehicles/LTEST000000000001/commands/"+command.CommandID, active.AccessToken, nil, 200, &command)
	}
	if command.Status == "pending" {
		t.Fatal("command was not dispatched")
	}
	// Real mTLS MQTT telemetry followed by QoS1 device ACKs closes the command loop.
	mqttTLS := tlsConfig.Clone()
	mqttTLS.NextProtos = []string{"mqtt"}
	mqttTLS.Certificates = []tls.Certificate{deviceCert}
	mqttConn, err := tls.Dial("tcp", "localhost:8883", mqttTLS)
	if err != nil {
		t.Fatal(err)
	}
	defer mqttConn.Close()
	if err := mqttConnect(mqttConn, "smoke-"+uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	mqttMessage := &vehiclev1.Telemetry{Envelope: &vehiclev1.Envelope{Version: "v1", MessageId: uuid.NewString(), Vin: "LTEST000000000001", DeviceBootId: "mqtt-" + uuid.NewString(), Sequence: 1, OccurredAtMs: time.Now().UnixMilli()}, BatteryPercent: 77}
	mqttRaw, _ := proto.Marshal(mqttMessage)
	if err := mqttPublish(mqttConn, "vehicles/LTEST000000000001/telemetry/v1", mqttRaw, 0, 1); err != nil {
		t.Fatal(err)
	}
	for _, ack := range []vehiclev1.CommandAck_Status{vehiclev1.CommandAck_STATUS_RECEIVED, vehiclev1.CommandAck_STATUS_SUCCEEDED} {
		message := &vehiclev1.CommandAck{Envelope: &vehiclev1.Envelope{Version: "v1", MessageId: uuid.NewString(), Vin: "LTEST000000000001", DeviceBootId: "ack-boot", OccurredAtMs: time.Now().UnixMilli()}, CommandId: command.CommandID, Status: ack}
		raw, _ := proto.Marshal(message)
		if err := mqttPublish(mqttConn, "vehicles/LTEST000000000001/ack/v1", raw, 1, uint16(ack)); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for command.Status != "succeeded" && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		s.json("GET", "/api/v1/vehicles/LTEST000000000001/commands/"+command.CommandID, active.AccessToken, nil, 200, &command)
	}
	if command.Status != "succeeded" {
		t.Fatalf("MQTT ACK did not finish command: %s", command.Status)
	}
	deadline = time.Now().Add(5 * time.Second)
	for latest.MessageID != mqttMessage.Envelope.MessageId && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		s.json("GET", "/api/v1/vehicles/LTEST000000000001/telemetry/latest", active.AccessToken, nil, 200, &latest)
	}
	if latest.MessageID != mqttMessage.Envelope.MessageId {
		t.Fatal("MQTT telemetry was not persisted")
	}
	var viewer tokenResponse
	s.json("POST", "/api/v1/auth/login", "", map[string]string{"username": "viewer", "password": "Viewer123!"}, 200, &viewer)
	var viewerVehicles struct {
		Items []domain.Vehicle `json:"items"`
	}
	s.json("GET", "/api/v1/vehicles", viewer.AccessToken, nil, 200, &viewerVehicles)
	if len(viewerVehicles.Items) != 1 {
		t.Fatal("viewer vehicle ABAC failed")
	}
	s.json("POST", "/api/v1/vehicles/LTEST000000000001/commands", viewer.AccessToken, request, 403, nil)
}
func (s *smokeClient) json(method, path, token string, input any, want int, output any) {
	s.t.Helper()
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			s.t.Fatal(err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, s.base+path, body)
	if err != nil {
		s.t.Fatal(err)
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := s.client.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		s.t.Fatal(err)
	}
	if response.StatusCode != want {
		s.t.Fatalf("%s %s: status %d want %d: %s", method, path, response.StatusCode, want, raw)
	}
	if output != nil {
		if err := json.Unmarshal(raw, output); err != nil {
			s.t.Fatalf("decode %s: %v", raw, err)
		}
	}
}
func Example_liveSmoke() { fmt.Println("CAR_INTEGRATION=1 go test ./test -v") }

func mqttConnect(conn net.Conn, clientID string) error {
	variable := []byte{0, 4, 'M', 'Q', 'T', 'T', 5, 2, 0, 30, 0}
	variable = append(variable, byte(len(clientID)>>8), byte(len(clientID)))
	variable = append(variable, clientID...)
	if err := mqttWrite(conn, 0x10, variable); err != nil {
		return err
	}
	header, payload, err := mqttRead(conn)
	if err != nil {
		return err
	}
	if header != 0x20 || len(payload) < 2 || payload[1] != 0 {
		return fmt.Errorf("MQTT CONNACK failed: %x %x", header, payload)
	}
	return nil
}

func mqttPublish(conn net.Conn, topic string, payload []byte, qos byte, packetID uint16) error {
	body := []byte{byte(len(topic) >> 8), byte(len(topic))}
	body = append(body, topic...)
	if qos > 0 {
		body = append(body, byte(packetID>>8), byte(packetID))
	}
	body = append(body, 0) // MQTT 5 property length
	body = append(body, payload...)
	if err := mqttWrite(conn, 0x30|(qos<<1), body); err != nil {
		return err
	}
	if qos == 1 {
		header, _, err := mqttRead(conn)
		if err != nil {
			return err
		}
		if header != 0x40 {
			return fmt.Errorf("expected PUBACK, got %x", header)
		}
	}
	return nil
}

func mqttWrite(conn net.Conn, header byte, body []byte) error {
	packet := []byte{header}
	length := len(body)
	for {
		digit := byte(length % 128)
		length /= 128
		if length > 0 {
			digit |= 128
		}
		packet = append(packet, digit)
		if length == 0 {
			break
		}
	}
	packet = append(packet, body...)
	_, err := conn.Write(packet)
	return err
}

func mqttRead(conn net.Conn) (byte, []byte, error) {
	var first [1]byte
	if _, err := io.ReadFull(conn, first[:]); err != nil {
		return 0, nil, err
	}
	length, multiplier := 0, 1
	for {
		var digit [1]byte
		if _, err := io.ReadFull(conn, digit[:]); err != nil {
			return 0, nil, err
		}
		length += int(digit[0]&127) * multiplier
		if digit[0]&128 == 0 {
			break
		}
		multiplier *= 128
		if multiplier > 128*128*128 {
			return 0, nil, fmt.Errorf("invalid MQTT length")
		}
	}
	payload := make([]byte, length)
	_, err := io.ReadFull(conn, payload)
	return first[0], payload, err
}

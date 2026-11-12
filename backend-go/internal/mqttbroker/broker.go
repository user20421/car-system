package mqttbroker

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/car-system/car-system/backend-go/internal/config"
	"github.com/car-system/car-system/backend-go/internal/domain"
	"github.com/car-system/car-system/backend-go/internal/realtime"
	"github.com/car-system/car-system/backend-go/internal/store"
	"github.com/car-system/car-system/backend-go/internal/telemetry"
	vehiclev1 "github.com/car-system/car-system/backend-go/pkg/vehiclev1"
	"github.com/google/uuid"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
	"google.golang.org/protobuf/proto"
)

type Broker struct {
	Server   *mqtt.Server
	store    *store.Store
	pipeline *telemetry.Pipeline
	hub      *realtime.Hub
	log      *slog.Logger
	sequence atomic.Uint64
	mu       sync.RWMutex
	boots    map[string]string
}

func New(cfg config.Config, db *store.Store, pipeline *telemetry.Pipeline, hub *realtime.Hub, log *slog.Logger) (*Broker, error) {
	server := mqtt.New(&mqtt.Options{InlineClient: true, Logger: log, Capabilities: mqtt.NewDefaultServerCapabilities()})
	b := &Broker{Server: server, store: db, pipeline: pipeline, hub: hub, log: log, boots: map[string]string{}}
	if err := server.AddHook(&vehicleHook{store: db, broker: b}, nil); err != nil {
		return nil, err
	}
	ca, err := os.ReadFile(cfg.CACertFile)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("invalid device CA")
	}
	cert, err := tls.LoadX509KeyPair(cfg.MQTTCertFile, cfg.MQTTKeyFile)
	if err != nil {
		return nil, err
	}
	base, err := net.Listen("tcp", cfg.MQTTAddr)
	if err != nil {
		return nil, err
	}
	listener := tls.NewListener(base, &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}, ClientCAs: roots, ClientAuth: tls.RequireAndVerifyClientCert, NextProtos: []string{"mqtt"}})
	if err := server.AddListener(listeners.NewNet("mqtts", listener)); err != nil {
		base.Close()
		return nil, err
	}
	return b, nil
}
func (b *Broker) Serve() error { return b.Server.Serve() }
func (b *Broker) Close() error { return b.Server.Close() }
func clientVIN(cl *mqtt.Client) string {
	if cl == nil || cl.Net.Conn == nil {
		return ""
	}
	conn, ok := cl.Net.Conn.(*tls.Conn)
	if !ok {
		return ""
	}
	state := conn.ConnectionState()
	if len(state.VerifiedChains) == 0 || len(state.PeerCertificates) == 0 {
		return ""
	}
	return state.PeerCertificates[0].Subject.CommonName
}
func allowedTopic(vin, topic string, write bool) bool {
	prefix := "vehicles/" + vin + "/"
	if !strings.HasPrefix(topic, prefix) || !strings.HasSuffix(topic, "/v1") {
		return false
	}
	kind := strings.TrimSuffix(strings.TrimPrefix(topic, prefix), "/v1")
	if write {
		return kind == "telemetry" || kind == "ack" || kind == "status"
	}
	return kind == "command" || kind == "status"
}

type vehicleHook struct {
	mqtt.HookBase
	store  *store.Store
	broker *Broker
}

func (h *vehicleHook) ID() string { return "vehicle-mtls-acl" }
func (h *vehicleHook) Provides(b byte) bool {
	return bytes.Contains([]byte{mqtt.OnConnectAuthenticate, mqtt.OnACLCheck, mqtt.OnPublished, mqtt.OnDisconnect}, []byte{b})
}
func (h *vehicleHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	vin := clientVIN(cl)
	if !domain.ValidVIN(vin) {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := h.store.VehicleByVIN(ctx, vin)
	return err == nil
}
func (h *vehicleHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	return allowedTopic(clientVIN(cl), topic, write)
}
func (h *vehicleHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	if cl.Net.Inline {
		return
	}
	vin := clientVIN(cl)
	if !domain.ValidVIN(vin) {
		return
	}
	payload := append([]byte(nil), pk.Payload...)
	topic := pk.TopicName
	// The broker invokes this hook on the client's read path. Keeping the
	// lightweight validation/enqueue work synchronous preserves ACK order.
	h.process(vin, topic, payload)
}
func (h *vehicleHook) process(vin, topic string, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	switch topic {
	case "vehicles/" + vin + "/telemetry/v1":
		t, err := telemetry.Decode(payload, vin, time.Now().UTC())
		if err != nil {
			h.broker.log.Warn("invalid telemetry", "vin", vin, "error", err)
			return
		}
		if err := h.broker.pipeline.Enqueue(ctx, t); err != nil {
			h.broker.log.Warn("telemetry dropped", "vin", vin, "error", err)
		}
	case "vehicles/" + vin + "/ack/v1":
		var ack vehiclev1.CommandAck
		if err := proto.Unmarshal(payload, &ack); err != nil || !validEnvelope(ack.Envelope, vin) {
			return
		}
		status := map[vehiclev1.CommandAck_Status]string{vehiclev1.CommandAck_STATUS_RECEIVED: "received", vehiclev1.CommandAck_STATUS_SUCCEEDED: "succeeded", vehiclev1.CommandAck_STATUS_FAILED: "failed", vehiclev1.CommandAck_STATUS_EXPIRED: "expired"}[ack.Status]
		if status == "" {
			return
		}
		var commandError *domain.CommandError
		if ack.Error != nil {
			commandError = &domain.CommandError{Code: ack.Error.Code, Message: ack.Error.Message, Retryable: ack.Error.Retryable, Details: ack.Error.Details}
		}
		command, changed, err := h.store.ApplyCommandStatus(ctx, ack.CommandId, status, commandError, time.Now().UnixMilli())
		if err != nil {
			h.broker.log.Warn("ACK rejected", "command_id", ack.CommandId, "error", err)
			return
		}
		if changed {
			h.broker.hub.Command(command)
		}
	case "vehicles/" + vin + "/status/v1":
		var presence vehiclev1.Presence
		if proto.Unmarshal(payload, &presence) != nil || !validEnvelope(presence.Envelope, vin) {
			return
		}
		h.broker.mu.Lock()
		h.broker.boots[vin] = presence.Envelope.DeviceBootId
		h.broker.mu.Unlock()
	}
}
func (h *vehicleHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	vin := clientVIN(cl)
	if !domain.ValidVIN(vin) {
		return
	}
	h.broker.mu.RLock()
	boot := h.broker.boots[vin]
	h.broker.mu.RUnlock()
	if boot == "" {
		boot = "unknown"
	}
	now := time.Now().UTC()
	message := &vehiclev1.Presence{Envelope: &vehiclev1.Envelope{Version: "v1", MessageId: uuid.NewString(), Vin: vin, DeviceBootId: boot, Sequence: h.broker.sequence.Add(1), OccurredAtMs: now.UnixMilli()}, State: vehiclev1.Presence_STATE_OFFLINE, ConnectionId: cl.ID}
	raw, _ := proto.Marshal(message)
	_ = h.broker.Server.Publish("vehicles/"+vin+"/status/v1", raw, true, 1)
}
func validEnvelope(e *vehiclev1.Envelope, vin string) bool {
	return e != nil && e.Version == "v1" && e.MessageId != "" && e.Vin == vin && e.DeviceBootId != "" && e.OccurredAtMs > 0
}

func (b *Broker) PublishCommand(ctx context.Context, c domain.Command) error {
	if c.ExpiresAtMS <= time.Now().UnixMilli() {
		return fmt.Errorf("command expired")
	}
	kind := map[string]vehiclev1.CommandType{"lock": vehiclev1.CommandType_COMMAND_TYPE_LOCK, "unlock": vehiclev1.CommandType_COMMAND_TYPE_UNLOCK, "hvac_on": vehiclev1.CommandType_COMMAND_TYPE_HVAC_ON, "hvac_off": vehiclev1.CommandType_COMMAND_TYPE_HVAC_OFF, "set_temperature": vehiclev1.CommandType_COMMAND_TYPE_SET_TEMPERATURE, "defrost_on": vehiclev1.CommandType_COMMAND_TYPE_DEFROST_ON, "defrost_off": vehiclev1.CommandType_COMMAND_TYPE_DEFROST_OFF, "honk_flash": vehiclev1.CommandType_COMMAND_TYPE_HONK_FLASH}[c.Type]
	if kind == 0 {
		return fmt.Errorf("unknown command type")
	}
	now := time.Now().UTC()
	message := &vehiclev1.Command{Envelope: &vehiclev1.Envelope{Version: "v1", MessageId: uuid.NewString(), Vin: c.VIN, DeviceBootId: "backend", Sequence: b.sequence.Add(1), OccurredAtMs: now.UnixMilli()}, CommandId: c.CommandID, IdempotencyKey: c.IdempotencyKey, ExpiresAtMs: c.ExpiresAtMS, Type: kind, Params: c.Params}
	raw, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	client, ok := b.Server.Clients.Get(mqtt.InlineClientId)
	if !ok {
		return fmt.Errorf("inline MQTT client unavailable")
	}
	expiry := uint32((time.Until(time.UnixMilli(c.ExpiresAtMS)) + time.Second - 1) / time.Second)
	packet := packets.Packet{FixedHeader: packets.FixedHeader{Type: packets.Publish, Qos: 1}, TopicName: "vehicles/" + c.VIN + "/command/v1", Payload: raw, PacketID: 1, Properties: packets.Properties{MessageExpiryInterval: expiry, ContentType: "application/x-protobuf", CorrelationData: []byte(c.CommandID)}}
	return b.Server.InjectPacket(client, packet)
}

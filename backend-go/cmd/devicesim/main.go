package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	vehiclev1 "github.com/car-system/car-system/backend-go/pkg/vehiclev1"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type simulator struct {
	vin        string
	bootID     string
	sequence   atomic.Uint64
	client     mqtt.Client
	locked     bool
	hvac       bool
	targetTemp float64
}

func main() {
	vin := flag.String("vin", "LTEST000000000001", "vehicle VIN")
	broker := flag.String("broker", "mqtts://localhost:8883", "MQTTS broker")
	certDir := flag.String("cert-dir", "../deploy/certs", "certificate directory")
	count := flag.Int("count", 0, "telemetry frames to send; 0 means continuous")
	interval := flag.Duration("interval", time.Second, "telemetry interval")
	flag.Parse()
	ca, err := os.ReadFile(filepath.Join(*certDir, "ca.crt"))
	must(err)
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		panic("invalid CA")
	}
	cert, err := tls.LoadX509KeyPair(filepath.Join(*certDir, *vin+".crt"), filepath.Join(*certDir, *vin+".key"))
	must(err)
	s := &simulator{vin: *vin, bootID: uuid.NewString(), locked: true, targetTemp: 22}
	opts := mqtt.NewClientOptions().AddBroker(*broker).SetClientID(*vin + "-" + uuid.NewString()).
		SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, Certificates: []tls.Certificate{cert}}).
		SetCleanSession(false).SetAutoReconnect(true).SetConnectRetry(true).SetKeepAlive(30 * time.Second).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) { fmt.Printf("connection lost: %v\n", err) })
	will := s.presence(vehiclev1.Presence_STATE_OFFLINE)
	opts.SetBinaryWill("vehicles/"+*vin+"/status/v1", will, 1, true)
	opts.OnConnect = func(client mqtt.Client) {
		fmt.Println("MQTTS connected")
		s.client = client
		mustToken(client.Publish("vehicles/"+s.vin+"/status/v1", 1, true, s.presence(vehiclev1.Presence_STATE_ONLINE)))
		mustToken(client.Subscribe("vehicles/"+s.vin+"/command/v1", 1, s.onCommand))
	}
	client := mqtt.NewClient(opts)
	s.client = client
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer client.Disconnect(500)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for sent := 0; *count == 0 || sent < *count; sent++ {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !client.IsConnectionOpen() {
				continue
			}
			mustToken(client.Publish("vehicles/"+s.vin+"/telemetry/v1", 0, false, s.telemetry(sent)))
			fmt.Printf("telemetry published: sequence=%d\n", s.sequence.Load())
		}
	}
}

func (s *simulator) envelope() *vehiclev1.Envelope {
	return &vehiclev1.Envelope{Version: "v1", MessageId: uuid.NewString(), Vin: s.vin, DeviceBootId: s.bootID, Sequence: s.sequence.Add(1), OccurredAtMs: time.Now().UnixMilli()}
}

func (s *simulator) presence(state vehiclev1.Presence_State) []byte {
	raw, err := proto.Marshal(&vehiclev1.Presence{Envelope: s.envelope(), State: state, ConnectionId: s.bootID, FirmwareVersion: "1.0.0-smoke"})
	must(err)
	return raw
}

func (s *simulator) telemetry(frame int) []byte {
	hvac := vehiclev1.HvacState_MODE_OFF
	if s.hvac {
		hvac = vehiclev1.HvacState_MODE_AUTO
	}
	message := &vehiclev1.Telemetry{
		Envelope: s.envelope(), SpeedKph: 35 + float64(frame%20), Rpm: 1800, BatteryPercent: 82, RangeKm: 438,
		Location: &vehiclev1.Location{Latitude: 31.2304, Longitude: 121.4737, HeadingDeg: 90}, Locked: s.locked,
		Hvac: &vehiclev1.HvacState{Mode: hvac, TargetTempC: s.targetTemp, FanLevel: 3, AcOn: s.hvac}, CabinTempC: 24,
	}
	raw, err := proto.Marshal(message)
	must(err)
	return raw
}

func (s *simulator) onCommand(_ mqtt.Client, message mqtt.Message) {
	var command vehiclev1.Command
	if err := proto.Unmarshal(message.Payload(), &command); err != nil {
		fmt.Printf("invalid command: %v\n", err)
		return
	}
	fmt.Printf("command received: id=%s type=%s\n", command.CommandId, command.Type)
	s.ack(command.CommandId, vehiclev1.CommandAck_STATUS_RECEIVED, nil)
	if command.ExpiresAtMs <= time.Now().UnixMilli() {
		s.ack(command.CommandId, vehiclev1.CommandAck_STATUS_EXPIRED, &vehiclev1.CommandError{Code: "COMMAND_EXPIRED", Message: "command expired"})
		return
	}
	switch command.Type {
	case vehiclev1.CommandType_COMMAND_TYPE_LOCK:
		s.locked = true
	case vehiclev1.CommandType_COMMAND_TYPE_UNLOCK:
		s.locked = false
	case vehiclev1.CommandType_COMMAND_TYPE_HVAC_ON:
		s.hvac = true
	case vehiclev1.CommandType_COMMAND_TYPE_HVAC_OFF:
		s.hvac = false
	case vehiclev1.CommandType_COMMAND_TYPE_SET_TEMPERATURE:
		if _, err := fmt.Sscan(command.Params["target_temp_c"], &s.targetTemp); err != nil {
			s.ack(command.CommandId, vehiclev1.CommandAck_STATUS_FAILED, &vehiclev1.CommandError{Code: "INVALID_PARAMETER", Message: err.Error()})
			return
		}
	default:
		s.ack(command.CommandId, vehiclev1.CommandAck_STATUS_FAILED, &vehiclev1.CommandError{Code: "UNSUPPORTED_COMMAND", Message: "smoke simulator does not support this command"})
		return
	}
	time.Sleep(100 * time.Millisecond)
	s.ack(command.CommandId, vehiclev1.CommandAck_STATUS_SUCCEEDED, nil)
	fmt.Printf("command succeeded: id=%s\n", command.CommandId)
}

func (s *simulator) ack(commandID string, status vehiclev1.CommandAck_Status, commandError *vehiclev1.CommandError) {
	raw, err := proto.Marshal(&vehiclev1.CommandAck{Envelope: s.envelope(), CommandId: commandID, Status: status, Error: commandError})
	must(err)
	mustToken(s.client.Publish("vehicles/"+s.vin+"/ack/v1", 1, false, raw))
}

func mustToken(token mqtt.Token) {
	if token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

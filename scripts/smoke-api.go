package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"
)

type tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type client struct {
	base          string
	http          *http.Client
	expectCommand string
}

func main() {
	base := flag.String("base", "https://localhost:8443", "backend base URL")
	caFile := flag.String("ca", "deploy/certs/ca.crt", "development CA")
	expectCommand := flag.String("expect-command", "sent", "expected command state: sent or succeeded")
	flag.Parse()
	ca, err := os.ReadFile(*caFile)
	must(err)
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		panic("invalid CA")
	}
	jar, err := cookiejar.New(nil)
	must(err)
	c := &client{base: *base, expectCommand: *expectCommand, http: &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}, ForceAttemptHTTP2: true},
		Jar:       jar,
	}}
	c.run()
}

func (c *client) run() {
	status, proto, body := c.request(http.MethodGet, "/health", "", nil)
	expect(status == http.StatusOK, "health", body)
	fmt.Printf("health: 200, protocol=%s\n", proto)
	admin := c.login("admin", "Admin123!")
	owner := c.login("owner", "Owner123!")
	viewer := c.login("viewer", "Viewer123!")
	c.vehicleCount("admin", admin.AccessToken, 3)
	c.vehicleCount("owner", owner.AccessToken, 2)
	c.vehicleCount("viewer", viewer.AccessToken, 1)

	commandBody := map[string]any{
		"idempotency_key": fmt.Sprintf("smoke-%d", time.Now().UnixNano()),
		"expires_at_ms":   time.Now().Add(2 * time.Minute).UnixMilli(),
		"type":            "lock",
		"params":          map[string]string{},
	}
	status, _, body = c.request(http.MethodPost, "/api/v1/vehicles/LTEST000000000001/commands", viewer.AccessToken, commandBody)
	expect(status == http.StatusForbidden, "viewer must not control a shared read-only vehicle", body)
	fmt.Println("viewer command isolation: 403")

	status, _, body = c.request(http.MethodPost, "/api/v1/vehicles/LTEST000000000001/commands", owner.AccessToken, commandBody)
	expect(status == http.StatusAccepted, "owner command", body)
	var created map[string]any
	must(json.Unmarshal(body, &created))
	fmt.Printf("owner command: 202, id=%v, status=%v\n", created["command_id"], created["status"])
	commandID, _ := created["command_id"].(string)
	expect(commandID != "", "command id missing", body)
	status, _, body = c.request(http.MethodPost, "/api/v1/vehicles/LTEST000000000001/commands", owner.AccessToken, commandBody)
	var duplicate map[string]any
	must(json.Unmarshal(body, &duplicate))
	expect(status == http.StatusAccepted && duplicate["command_id"] == commandID, "idempotent command replay", body)
	fmt.Println("command idempotency: OK")
	commandPath := "/api/v1/vehicles/LTEST000000000001/commands/" + commandID
	var commandStatus string
	for range 50 {
		status, _, body = c.request(http.MethodGet, commandPath, owner.AccessToken, nil)
		expect(status == http.StatusOK, "query command", body)
		var current map[string]any
		must(json.Unmarshal(body, &current))
		commandStatus, _ = current["status"].(string)
		if commandStatus == c.expectCommand {
			break
		}
		expect(commandStatus != "failed" && commandStatus != "expired", "command reached unexpected terminal state", body)
		time.Sleep(100 * time.Millisecond)
	}
	expect(commandStatus == c.expectCommand || (c.expectCommand == "sent" && commandStatus == "succeeded"), "RabbitMQ/MQTT dispatch did not reach "+c.expectCommand+"; final status="+commandStatus, body)
	fmt.Println("RabbitMQ outbox and MQTT dispatch:", commandStatus)
	if c.expectCommand == "succeeded" {
		status, _, body = c.request(http.MethodGet, "/api/v1/vehicles/LTEST000000000001/telemetry/latest", owner.AccessToken, nil)
		expect(status == http.StatusOK && bytes.Contains(body, []byte("LTEST000000000001")), "latest MQTTS telemetry", body)
		fmt.Println("MQTTS Protobuf telemetry and device ACK: OK")
	}

	status, _, body = c.request(http.MethodPost, "/api/v1/ws-ticket", owner.AccessToken, map[string]any{})
	expect(status == http.StatusCreated, "WSS ticket", body)
	fmt.Println("WSS ticket: 201")

	status, _, body = c.request(http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{"refresh_token": owner.RefreshToken})
	expect(status == http.StatusOK, "refresh rotation", body)
	status, _, body = c.request(http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{"refresh_token": owner.RefreshToken})
	expect(status == http.StatusUnauthorized, "refresh token reuse must be rejected", body)
	fmt.Println("refresh rotation and reuse detection: OK")
	// Reuse detection intentionally revokes the whole session family. Create a
	// fresh session before validating browser cookie restoration.
	owner = c.login("owner", "Owner123!")
	status, _, body = c.request(http.MethodPost, "/api/v1/auth/refresh", "", nil)
	expect(status == http.StatusOK, "browser cookie session restore", body)
	fmt.Println("HttpOnly browser session restore: OK")
	fmt.Println("API smoke test passed")
}

func (c *client) login(username, password string) tokens {
	status, _, body := c.request(http.MethodPost, "/api/v1/auth/login", "", map[string]string{"username": username, "password": password})
	expect(status == http.StatusOK, username+" login", body)
	var result tokens
	must(json.Unmarshal(body, &result))
	return result
}

func (c *client) vehicleCount(name, token string, want int) {
	status, _, body := c.request(http.MethodGet, "/api/v1/vehicles", token, nil)
	expect(status == http.StatusOK, name+" vehicles", body)
	var result struct {
		Items []any `json:"items"`
	}
	must(json.Unmarshal(body, &result))
	expect(len(result.Items) == want, fmt.Sprintf("%s vehicle count: got %d want %d", name, len(result.Items), want), body)
	fmt.Printf("%s accessible vehicles: %d\n", name, len(result.Items))
}

func (c *client) request(method, path, token string, input any) (int, string, []byte) {
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		must(err)
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.base+path, body)
	must(err)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Request-ID", fmt.Sprintf("smoke-%d", time.Now().UnixNano()))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	must(err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	must(err)
	return resp.StatusCode, resp.Proto, raw
}

func expect(ok bool, message string, body []byte) {
	if !ok {
		panic(fmt.Sprintf("%s: %s", message, body))
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment                                      string
	HTTPAddr                                         string
	MQTTAddr                                         string
	PublicURL                                        string
	TLSDir                                           string
	CACertFile                                       string
	HTTPCertFile, HTTPKeyFile                        string
	MQTTCertFile, MQTTKeyFile                        string
	JWTKeyFile                                       string
	MySQL                                            MySQL
	Redis                                            Redis
	RabbitMQ                                         RabbitMQ
	TelemetryWorkers, TelemetryQueue, TelemetryBatch int
}

type MySQL struct{ Host, Port, Database, User, Password string }
type Redis struct{ Addr, User, Password string }
type RabbitMQ struct{ URL string }

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}
	dir := env("TLS_DIR", "../deploy/certs")
	c := Config{
		Environment: env("APP_ENV", "development"), HTTPAddr: env("HTTPS_ADDR", ":8443"), MQTTAddr: env("MQTTS_ADDR", ":8883"),
		PublicURL: env("PUBLIC_URL", "https://localhost:8443"), TLSDir: dir,
		CACertFile:   env("CA_CERT_FILE", filepath.Join(dir, "ca.crt")),
		HTTPCertFile: env("HTTPS_CERT_FILE", filepath.Join(dir, "https-server.crt")), HTTPKeyFile: env("HTTPS_KEY_FILE", filepath.Join(dir, "https-server.key")),
		MQTTCertFile: env("MQTTS_CERT_FILE", filepath.Join(dir, "mqtt-server.crt")), MQTTKeyFile: env("MQTTS_KEY_FILE", filepath.Join(dir, "mqtt-server.key")),
		JWTKeyFile:       env("JWT_PRIVATE_KEY_FILE", ".secrets/jwt-es256.key"),
		MySQL:            MySQL{env("MYSQL_HOST", "localhost"), env("MYSQL_PORT", "3306"), env("MYSQL_DATABASE", "car_system"), env("MYSQL_USER", "car_app"), env("MYSQL_PASSWORD", "")},
		Redis:            Redis{env("REDIS_ADDR", "localhost:6379"), env("REDIS_USER", "car_app"), env("REDIS_PASSWORD", "")},
		RabbitMQ:         RabbitMQ{env("RABBITMQ_URL", "")},
		TelemetryWorkers: intEnv("TELEMETRY_WORKERS", 4), TelemetryQueue: intEnv("TELEMETRY_QUEUE", 4096), TelemetryBatch: intEnv("TELEMETRY_BATCH_SIZE", 200),
	}
	if c.RabbitMQ.URL == "" {
		user, pass := env("RABBITMQ_USER", "car_app"), os.Getenv("RABBITMQ_PASSWORD")
		host := env("RABBITMQ_HOST", "localhost:5671")
		c.RabbitMQ.URL = (&url.URL{Scheme: "amqps", User: url.UserPassword(user, pass), Host: host, Path: "/car"}).String()
	}
	if c.MySQL.Password == "" || c.Redis.Password == "" || os.Getenv("RABBITMQ_PASSWORD") == "" && !strings.Contains(c.RabbitMQ.URL, "@") {
		return c, fmt.Errorf("MYSQL_PASSWORD, REDIS_PASSWORD and RABBITMQ_PASSWORD are required")
	}
	if !strings.HasPrefix(c.RabbitMQ.URL, "amqps://") || !strings.HasPrefix(c.PublicURL, "https://") {
		return c, fmt.Errorf("RabbitMQ and public endpoint must use TLS")
	}
	for name, addr := range map[string]string{"HTTPS_ADDR": c.HTTPAddr, "MQTTS_ADDR": c.MQTTAddr} {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			return c, fmt.Errorf("%s: %w", name, err)
		}
	}
	if c.TelemetryWorkers < 1 || c.TelemetryWorkers > 32 || c.TelemetryQueue < 1 || c.TelemetryBatch < 1 || c.TelemetryBatch > 1000 {
		return c, fmt.Errorf("invalid telemetry worker, queue or batch configuration")
	}
	for _, file := range []string{c.CACertFile, c.HTTPCertFile, c.HTTPKeyFile, c.MQTTCertFile, c.MQTTKeyFile} {
		if _, err := os.Stat(file); err != nil {
			return c, fmt.Errorf("TLS file %s: %w", file, err)
		}
	}
	return c, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func intEnv(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value == 0 {
		return fallback
	}
	return value
}
func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for lineNumber, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return fmt.Errorf("%s:%d: expected KEY=VALUE", path, lineNumber+1)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

// LoadOrCreateJWTKey keeps the ES256 private key on disk, never in MySQL. Production
// deployments should provision this file through their secret manager.
func LoadOrCreateJWTKey(path string) (*ecdsa.PrivateKey, error) {
	if data, err := os.ReadFile(path); err == nil {
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("invalid JWT PEM key")
		}
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil || key.Curve != elliptic.P256() {
			return nil, fmt.Errorf("JWT key must be an ES256 EC private key: %w", err)
		}
		return key, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if os.Getenv("APP_ENV") == "production" {
		return nil, fmt.Errorf("JWT_PRIVATE_KEY_FILE does not exist in production")
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

const AccessTokenTTL = 15 * time.Minute

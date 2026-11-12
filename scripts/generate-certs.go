// generate-certs.go creates a development-only PKI using only the Go standard library.
// Run from the repository root: go run ./scripts/generate-certs.go
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type certificateSpec struct {
	name       string
	commonName string
	dnsNames   []string
	client     bool
}

func main() {
	out := flag.String("out", "deploy/certs", "output directory")
	days := flag.Int("days", 825, "leaf certificate validity in days")
	caDays := flag.Int("ca-days", 3650, "CA validity in days")
	flag.Parse()
	if *days < 1 || *caDays < 1 {
		fatal("validity periods must be positive")
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fatal("create output directory: %v", err)
	}

	now := time.Now().UTC()
	caKey, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		fatal("generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          serial(),
		Subject:               pkix.Name{CommonName: "car-system development CA", Organization: []string{"car-system local development"}},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.AddDate(0, 0, *caDays),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		fatal("create CA certificate: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		fatal("parse CA certificate: %v", err)
	}
	writePEM(filepath.Join(*out, "ca.crt"), "CERTIFICATE", caDER, 0o644)
	writeKey(filepath.Join(*out, "ca.key"), caKey)

	localNames := []string{"localhost", "host.docker.internal"}
	specs := []certificateSpec{
		{name: "https-server", commonName: "car-system HTTPS", dnsNames: append(localNames, "api", "backend-go")},
		{name: "mqtt-server", commonName: "car-system MQTT", dnsNames: append(localNames, "rabbitmq", "mqtt")},
		{name: "redis-server", commonName: "car-system Redis", dnsNames: append(localNames, "redis")},
		{name: "rabbitmq-server", commonName: "car-system RabbitMQ", dnsNames: append(localNames, "rabbitmq")},
		{name: "mysql-server", commonName: "car-system MySQL", dnsNames: append(localNames, "mysql")},
		{name: "LTEST000000000001", commonName: "LTEST000000000001", client: true},
		{name: "LTEST000000000002", commonName: "LTEST000000000002", client: true},
		{name: "LTEST000000000003", commonName: "LTEST000000000003", client: true},
	}
	for _, spec := range specs {
		issue(*out, spec, caCert, caKey, now, *days)
		fmt.Printf("generated %s.crt and %s.key\n", spec.name, spec.name)
	}
	fmt.Printf("development CA and %d leaf certificates written to %s\n", len(specs), *out)
}

func issue(out string, spec certificateSpec, ca *x509.Certificate, caKey *rsa.PrivateKey, now time.Time, days int) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fatal("generate %s key: %v", spec.name, err)
	}
	template := &x509.Certificate{
		SerialNumber: serial(),
		Subject: pkix.Name{
			CommonName:   spec.commonName,
			Organization: []string{"car-system local development"},
		},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.AddDate(0, 0, days),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
		DNSNames:              spec.dnsNames,
	}
	if spec.client {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		template.URIs = []*url.URL{{Scheme: "urn", Opaque: "vehicle:vin:" + spec.commonName}}
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		fatal("create %s certificate: %v", spec.name, err)
	}
	writePEM(filepath.Join(out, spec.name+".crt"), "CERTIFICATE", der, 0o644)
	writeKey(filepath.Join(out, spec.name+".key"), key)
}

func serial() *big.Int {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		fatal("generate certificate serial: %v", err)
	}
	return value
}

func writePEM(path, blockType string, der []byte, mode os.FileMode) {
	data := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	if err := os.WriteFile(path, data, mode); err != nil {
		fatal("write %s: %v", path, err)
	}
}

func writeKey(path string, key *rsa.PrivateKey) {
	writePEM(path, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key), 0o600)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

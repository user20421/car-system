package store

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/car-system/car-system/backend-go/internal/auth"
	"github.com/car-system/car-system/backend-go/internal/config"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Store struct{ DB *sql.DB }

func Open(ctx context.Context, cfg config.Config) (*Store, error) {
	ca, err := os.ReadFile(cfg.CACertFile)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("invalid CA certificate")
	}
	host, _, _ := net.SplitHostPort(net.JoinHostPort(cfg.MySQL.Host, cfg.MySQL.Port))
	tlsName := "car-mysql-" + uuid.NewString()
	if err := mysql.RegisterTLSConfig(tlsName, &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: host}); err != nil {
		return nil, err
	}
	dsn := mysql.Config{User: cfg.MySQL.User, Passwd: cfg.MySQL.Password, Net: "tcp", Addr: net.JoinHostPort(cfg.MySQL.Host, cfg.MySQL.Port), DBName: cfg.MySQL.Database, TLSConfig: tlsName, ParseTime: true, Loc: time.UTC, Timeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}
	db, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("MySQL TLS ping: %w", err)
	}
	return &Store{DB: db}, nil
}

func (s *Store) Close() error                   { return s.DB.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.DB.PingContext(ctx) }

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(255) PRIMARY KEY, checksum CHAR(64) NOT NULL, applied_at DATETIME(6) NOT NULL) ENGINE=InnoDB`); err != nil {
		return err
	}
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		data, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])
		var existing string
		err = s.DB.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE version=?", entry.Name()).Scan(&existing)
		if err == nil {
			if existing != checksum {
				return fmt.Errorf("migration %s checksum changed", entry.Name())
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		for _, statement := range strings.Split(string(data), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err := s.DB.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migration %s: %w", entry.Name(), err)
			}
		}
		if _, err := s.DB.ExecContext(ctx, "INSERT INTO schema_migrations(version,checksum,applied_at) VALUES(?,?,UTC_TIMESTAMP(6))", entry.Name(), checksum); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Seed(ctx context.Context) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	hasher := auth.DefaultArgon2id()
	users := []struct {
		id, username, password, display string
		roles                           string
	}{
		{"00000000-0000-0000-0000-000000000001", "admin", "Admin123!", "Administrator", `["user","admin"]`},
		{"00000000-0000-0000-0000-000000000002", "owner", "Owner123!", "Vehicle Owner", `["user"]`},
		{"00000000-0000-0000-0000-000000000003", "viewer", "Viewer123!", "Vehicle Viewer", `["user"]`},
	}
	for _, u := range users {
		var exists int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id=?", u.id).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		hash, err := hasher.Hash(u.password)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO users(id,username,email,display_name,roles,password_hash,created_at) VALUES(?,?,?,?,?,?,UTC_TIMESTAMP(6))", u.id, u.username, u.username+"@example.local", u.display, u.roles, hash); err != nil {
			return err
		}
	}
	now := time.Now().UTC().UnixMilli()
	for i, owner := range []string{users[1].id, users[1].id, users[0].id} {
		vin := fmt.Sprintf("LTEST000000000%03d", i+1)
		if _, err := tx.ExecContext(ctx, `INSERT INTO vehicles(vin,display_name,model,model_year,owner_user_id,firmware_version,created_at) VALUES(?,?,?,?,?,?,UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE vin=VALUES(vin)`, vin, fmt.Sprintf("Test Vehicle %d", i+1), "Car Test", 2026, owner, "1.0.0"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO device_configs(vin,config_version,telemetry_interval_ms,config_poll_interval_ms,updated_at_ms) VALUES(?,1,1000,60000,?) ON DUPLICATE KEY UPDATE vin=VALUES(vin)`, vin, now); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO vehicle_shares(vin,user_id,access_role,created_at) VALUES('LTEST000000000001',?,'viewer',UTC_TIMESTAMP(6)) ON DUPLICATE KEY UPDATE access_role=VALUES(access_role)`, users[2].id); err != nil {
		return err
	}
	return tx.Commit()
}

package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/car-system/car-system/backend-go/internal/domain"
	"github.com/google/uuid"
)

func insertTelemetry(ctx context.Context, tx *sql.Tx, t domain.Telemetry) (bool, error) {
	payload, err := json.Marshal(t)
	if err != nil {
		return false, err
	}
	res, err := tx.ExecContext(ctx, `INSERT IGNORE INTO telemetry(message_id,vin,device_boot_id,sequence,occurred_at_ms,received_at_ms,payload) VALUES(?,?,?,?,?,?,?)`, t.MessageID, t.VIN, t.DeviceBootID, t.Sequence, t.OccurredAtMS, t.ReceivedAtMS, payload)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	for _, f := range t.Fault {
		if !f.Active {
			continue
		}
		title := f.Message
		if title == "" {
			title = f.Code
		}
		if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO alerts(id,vin,code,title,severity,occurred_at_ms) VALUES(?,?,?,?,?,?)`, uuid.NewString(), t.VIN, f.Code, title, f.Severity, t.OccurredAtMS); err != nil {
			return false, err
		}
	}
	return true, nil
}
func (s *Store) InsertTelemetryBatch(ctx context.Context, items []domain.Telemetry) (accepted []domain.Telemetry, duplicates int, err error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	for _, t := range items {
		ok, err := insertTelemetry(ctx, tx, t)
		if err != nil {
			return nil, 0, err
		}
		if ok {
			accepted = append(accepted, t)
		} else {
			duplicates++
		}
	}
	return accepted, duplicates, tx.Commit()
}
func (s *Store) InsertOfflineBatch(ctx context.Context, vin, batchID string, items []domain.Telemetry, now int64) (int, int, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "INSERT IGNORE INTO telemetry_batches(batch_id,vin,created_at_ms) VALUES(?,?,?)", batchID, vin, now)
	if err != nil {
		return 0, 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, 0, err
	}
	if n == 0 {
		return 0, len(items), nil
	}
	accepted := 0
	duplicates := 0
	for _, t := range items {
		ok, err := insertTelemetry(ctx, tx, t)
		if err != nil {
			return 0, 0, err
		}
		if ok {
			accepted++
		} else {
			duplicates++
		}
	}
	return accepted, duplicates, tx.Commit()
}
func (s *Store) LatestTelemetry(ctx context.Context, vin string) (domain.Telemetry, error) {
	var data []byte
	err := s.DB.QueryRowContext(ctx, "SELECT payload FROM telemetry WHERE vin=? ORDER BY occurred_at_ms DESC,id DESC LIMIT 1", vin).Scan(&data)
	if err == sql.ErrNoRows {
		return domain.Telemetry{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Telemetry{}, err
	}
	var t domain.Telemetry
	err = json.Unmarshal(data, &t)
	return t, err
}

type historyCursor struct {
	Occurred int64  `json:"o"`
	ID       uint64 `json:"i"`
}

func (s *Store) TelemetryHistory(ctx context.Context, vin string, from, to int64, limit int, cursor string) ([]domain.Telemetry, *string, error) {
	where := "vin=? AND occurred_at_ms>=? AND occurred_at_ms<=?"
	args := []any{vin, from, to}
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: cursor", domain.ErrInvalid)
		}
		var c historyCursor
		if json.Unmarshal(raw, &c) != nil {
			return nil, nil, fmt.Errorf("%w: cursor", domain.ErrInvalid)
		}
		where += " AND (occurred_at_ms>? OR (occurred_at_ms=? AND id>?))"
		args = append(args, c.Occurred, c.Occurred, c.ID)
	}
	args = append(args, limit+1)
	rows, err := s.DB.QueryContext(ctx, "SELECT id,occurred_at_ms,payload FROM telemetry WHERE "+where+" ORDER BY occurred_at_ms,id LIMIT ?", args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []domain.Telemetry{}
	var last historyCursor
	more := false
	for rows.Next() {
		var data []byte
		var current historyCursor
		if err := rows.Scan(&current.ID, &current.Occurred, &data); err != nil {
			return nil, nil, err
		}
		if len(items) == limit {
			more = true
			continue
		}
		var t domain.Telemetry
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, nil, err
		}
		items = append(items, t)
		last = current
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var next *string
	if more {
		raw, _ := json.Marshal(last)
		encoded := base64.RawURLEncoding.EncodeToString(raw)
		next = &encoded
	}
	return items, next, nil
}

func ValidateTimeRange(from, to int64) error {
	if from <= 0 || to < from || to-from > 31*24*60*60*1000 {
		return fmt.Errorf("%w: time range must be positive, ordered and at most 31 days", domain.ErrInvalid)
	}
	return nil
}
func ParseLimit(raw string, def, max int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		return 0, fmt.Errorf("%w: limit", domain.ErrInvalid)
	}
	return n, nil
}

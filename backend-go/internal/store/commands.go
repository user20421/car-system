package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

func scanCommand(scanner interface{ Scan(...any) error }) (domain.Command, error) {
	var c domain.Command
	var params, commandError []byte
	var errText sql.NullString
	err := scanner.Scan(&c.CommandID, &c.VIN, &c.RequestedBy, &c.IdempotencyKey, &c.ExpiresAtMS, &c.Type, &params, &c.Status, &commandError, &c.CreatedAtMS, &c.UpdatedAtMS)
	if err == sql.ErrNoRows {
		return c, domain.ErrNotFound
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(params, &c.Params); err != nil {
		return c, err
	}
	if len(commandError) > 0 && string(commandError) != "null" {
		if err := json.Unmarshal(commandError, &c.Error); err != nil {
			return c, err
		}
	}
	_ = errText
	return c, nil
}

const commandColumns = "command_id,vin,requested_by,idempotency_key,expires_at_ms,type,params,status,error,created_at_ms,updated_at_ms"

func (s *Store) CommandForUser(ctx context.Context, user domain.User, vin, id string) (domain.Command, error) {
	c, err := scanCommand(s.DB.QueryRowContext(ctx, "SELECT "+commandColumns+" FROM commands WHERE command_id=? AND vin=?", id, vin))
	if err != nil {
		return c, err
	}
	if _, err := s.VehicleForUser(ctx, user, vin); err != nil {
		return domain.Command{}, err
	}
	return c, nil
}
func (s *Store) CommandByID(ctx context.Context, id string) (domain.Command, error) {
	return scanCommand(s.DB.QueryRowContext(ctx, "SELECT "+commandColumns+" FROM commands WHERE command_id=?", id))
}

func (s *Store) CreateCommand(ctx context.Context, user domain.User, vin, key, kind string, params map[string]string, expires int64) (domain.Command, error) {
	if _, err := s.VehicleForUser(ctx, user, vin); err != nil {
		return domain.Command{}, err
	}
	// VehicleForUser permits viewers to read, but not to mutate the vehicle.
	v, err := s.VehicleForUser(ctx, user, vin)
	if err != nil {
		return domain.Command{}, err
	}
	if !domain.CanControl(user, v, v.AccessRole) {
		return domain.Command{}, domain.ErrForbidden
	}
	canonical := domain.CanonicalCommand(kind, params, expires)
	hash := sha256.Sum256([]byte(canonical))
	now := time.Now().UTC().UnixMilli()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return domain.Command{}, err
	}
	defer tx.Rollback()
	existing, err := scanCommand(tx.QueryRowContext(ctx, "SELECT "+commandColumns+" FROM commands WHERE requested_by=? AND vin=? AND idempotency_key=? FOR UPDATE", user.ID, vin, key))
	if err == nil {
		var oldHash []byte
		if err := tx.QueryRowContext(ctx, "SELECT request_hash FROM commands WHERE command_id=?", existing.CommandID).Scan(&oldHash); err != nil {
			return domain.Command{}, err
		}
		if string(oldHash) != string(hash[:]) {
			return domain.Command{}, fmt.Errorf("%w: idempotency key reused with different request", domain.ErrConflict)
		}
		return existing, tx.Commit()
	}
	if err != domain.ErrNotFound {
		return domain.Command{}, err
	}
	c := domain.Command{CommandID: uuid.NewString(), VIN: vin, RequestedBy: user.ID, IdempotencyKey: key, ExpiresAtMS: expires, Type: kind, Params: params, Status: "pending", CreatedAtMS: now, UpdatedAtMS: now}
	paramsJSON, _ := json.Marshal(params)
	payload, _ := json.Marshal(c)
	eventID := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `INSERT INTO commands(command_id,vin,requested_by,idempotency_key,request_hash,expires_at_ms,type,params,status,created_at_ms,updated_at_ms) VALUES(?,?,?,?,?,?,?,?, 'pending',?,?)`, c.CommandID, vin, user.ID, key, hash[:], expires, kind, paramsJSON, now, now); err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			tx.Rollback()
			return s.CreateCommand(ctx, user, vin, key, kind, params, expires)
		}
		return domain.Command{}, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO outbox(id,event_type,aggregate_id,payload,created_at_ms) VALUES(?,?,?,?,?)", eventID, "CommandCreated", c.CommandID, payload, now); err != nil {
		return domain.Command{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Command{}, err
	}
	return c, nil
}

func (s *Store) ApplyCommandStatus(ctx context.Context, id, status string, commandError *domain.CommandError, now int64) (domain.Command, bool, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return domain.Command{}, false, err
	}
	defer tx.Rollback()
	c, err := scanCommand(tx.QueryRowContext(ctx, "SELECT "+commandColumns+" FROM commands WHERE command_id=? FOR UPDATE", id))
	if err != nil {
		return c, false, err
	}
	if c.Status == status {
		return c, false, tx.Commit()
	}
	if c.Status == "succeeded" || c.Status == "failed" || c.Status == "expired" {
		return c, false, tx.Commit()
	}
	next, err := domain.Transition(c.Status, status)
	if err != nil {
		return c, false, err
	}
	var raw any
	if commandError != nil {
		raw, _ = json.Marshal(commandError)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE commands SET status=?,error=?,updated_at_ms=? WHERE command_id=?", next, raw, now, id); err != nil {
		return c, false, err
	}
	c.Status = next
	c.Error = commandError
	c.UpdatedAtMS = now
	return c, true, tx.Commit()
}
func (s *Store) ExpireCommands(ctx context.Context, now int64, limit int) ([]domain.Command, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+commandColumns+" FROM commands WHERE expires_at_ms<=? AND status IN ('pending','sent','received') LIMIT ?", now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	commands := []domain.Command{}
	for rows.Next() {
		c, err := scanCommand(rows)
		if err != nil {
			return nil, err
		}
		commands = append(commands, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := []domain.Command{}
	for _, c := range commands {
		updated, changed, err := s.ApplyCommandStatus(ctx, c.CommandID, "expired", &domain.CommandError{Code: "COMMAND_EXPIRED", Message: "command expired", Retryable: false}, now)
		if err == nil && changed {
			result = append(result, updated)
		}
	}
	return result, nil
}

type OutboxEvent struct {
	ID, Type, AggregateID string
	Payload               []byte
	CreatedAtMS           int64
}

func (s *Store) PublishOutbox(ctx context.Context, limit int, publish func(context.Context, OutboxEvent) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT id,event_type,aggregate_id,payload,created_at_ms FROM outbox WHERE published_at_ms IS NULL ORDER BY created_at_ms LIMIT ? FOR UPDATE SKIP LOCKED", limit)
	if err != nil {
		return err
	}
	events := []OutboxEvent{}
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.Type, &e.AggregateID, &e.Payload, &e.CreatedAtMS); err != nil {
			rows.Close()
			return err
		}
		events = append(events, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, e := range events {
		if err := publish(ctx, e); err != nil {
			_, _ = tx.ExecContext(ctx, "UPDATE outbox SET attempts=attempts+1,last_error=? WHERE id=?", err.Error(), e.ID)
			// Stop at the first failure so a later command cannot overtake it.
			break
		}
		if _, err := tx.ExecContext(ctx, "UPDATE outbox SET published_at_ms=? WHERE id=?", time.Now().UTC().UnixMilli(), e.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) DispatchProcessed(ctx context.Context, eventID string) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM command_dispatch WHERE event_id=?", eventID).Scan(&n)
	return n > 0, err
}
func (s *Store) RecordDispatch(ctx context.Context, eventID, commandID string, now int64) (bool, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "INSERT IGNORE INTO command_dispatch(command_id,event_id,processed_at_ms) VALUES(?,?,?)", commandID, eventID, now)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, tx.Commit()
	}
	if _, err := tx.ExecContext(ctx, "UPDATE commands SET status='sent',updated_at_ms=? WHERE command_id=? AND status='pending'", now, commandID); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

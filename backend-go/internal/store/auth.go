package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/car-system/car-system/backend-go/internal/auth"
	"github.com/car-system/car-system/backend-go/internal/domain"
	"github.com/google/uuid"
)

func scanUser(scanner interface{ Scan(...any) error }) (domain.User, error) {
	var u domain.User
	var roles []byte
	var email sql.NullString
	err := scanner.Scan(&u.ID, &u.Username, &email, &u.DisplayName, &roles, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return u, domain.ErrNotFound
	}
	if err != nil {
		return u, err
	}
	u.Email = email.String
	if err := json.Unmarshal(roles, &u.Roles); err != nil {
		return u, err
	}
	return u, nil
}

const userColumns = "id,username,email,display_name,roles,password_hash,created_at"

func (s *Store) UserByUsername(ctx context.Context, username string) (domain.User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE username=?", username))
}
func (s *Store) UserByID(ctx context.Context, id string) (domain.User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id=?", id))
}

func (s *Store) CreateSession(ctx context.Context, sid, uid, hash string, expires time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "INSERT INTO sessions(id,user_id,created_at) VALUES(?,?,UTC_TIMESTAMP(6))", sid, uid); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO refresh_tokens(id,session_id,user_id,token_hash,expires_at,created_at) VALUES(?,?,?,?,?,UTC_TIMESTAMP(6))", uuid.NewString(), sid, uid, hash, expires); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) CreateSessionRefresh(ctx context.Context, sid, uid, hash string, expires time.Time) error {
	_, err := s.DB.ExecContext(ctx, "INSERT INTO refresh_tokens(id,session_id,user_id,token_hash,expires_at,created_at) VALUES(?,?,?,?,?,UTC_TIMESTAMP(6))", uuid.NewString(), sid, uid, hash, expires)
	return err
}
func (s *Store) SessionActive(ctx context.Context, sid, uid string) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions WHERE id=? AND user_id=? AND revoked_at IS NULL", sid, uid).Scan(&n)
	return n == 1, err
}
func (s *Store) RevokeSession(ctx context.Context, sid string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "UPDATE sessions SET revoked_at=UTC_TIMESTAMP(6) WHERE id=? AND revoked_at IS NULL", sid); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE refresh_tokens SET revoked_at=UTC_TIMESTAMP(6) WHERE session_id=? AND revoked_at IS NULL", sid); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) RotateRefresh(ctx context.Context, hash string, now time.Time) (auth.Rotation, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return auth.Rotation{}, err
	}
	defer tx.Rollback()
	var id, sid, uid string
	var used, revoked sql.NullTime
	var expires time.Time
	err = tx.QueryRowContext(ctx, "SELECT id,session_id,user_id,expires_at,used_at,revoked_at FROM refresh_tokens WHERE token_hash=? FOR UPDATE", hash).Scan(&id, &sid, &uid, &expires, &used, &revoked)
	if err == sql.ErrNoRows {
		return auth.Rotation{}, domain.ErrUnauthorized
	}
	if err != nil {
		return auth.Rotation{}, err
	}
	if used.Valid || revoked.Valid {
		_, _ = tx.ExecContext(ctx, "UPDATE sessions SET revoked_at=UTC_TIMESTAMP(6) WHERE id=?", sid)
		_, _ = tx.ExecContext(ctx, "UPDATE refresh_tokens SET revoked_at=UTC_TIMESTAMP(6) WHERE session_id=? AND revoked_at IS NULL", sid)
		if commitErr := tx.Commit(); commitErr != nil {
			return auth.Rotation{}, commitErr
		}
		return auth.Rotation{}, auth.ErrTokenReused
	}
	if !expires.After(now) {
		return auth.Rotation{}, domain.ErrUnauthorized
	}
	var sessionRevoked sql.NullTime
	if err := tx.QueryRowContext(ctx, "SELECT revoked_at FROM sessions WHERE id=? FOR UPDATE", sid).Scan(&sessionRevoked); err != nil {
		return auth.Rotation{}, domain.ErrUnauthorized
	}
	if sessionRevoked.Valid {
		return auth.Rotation{}, domain.ErrUnauthorized
	}
	if _, err := tx.ExecContext(ctx, "UPDATE refresh_tokens SET used_at=UTC_TIMESTAMP(6) WHERE id=? AND used_at IS NULL", id); err != nil {
		return auth.Rotation{}, err
	}
	user, err := scanUser(tx.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id=?", uid))
	if err != nil {
		return auth.Rotation{}, err
	}
	if err := tx.Commit(); err != nil {
		return auth.Rotation{}, err
	}
	return auth.Rotation{User: user, SessionID: sid, TokenID: id}, nil
}

package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
	"github.com/google/uuid"
)

type memoryToken struct {
	sessionID, userID, hash string
	expires                 time.Time
	used, revoked           bool
}
type memoryStore struct {
	user     domain.User
	sessions map[string]bool
	tokens   map[string]*memoryToken
}

func newMemoryStore(u domain.User) *memoryStore {
	return &memoryStore{user: u, sessions: map[string]bool{}, tokens: map[string]*memoryToken{}}
}
func (m *memoryStore) UserByUsername(context.Context, string) (domain.User, error) {
	return m.user, nil
}
func (m *memoryStore) UserByID(context.Context, string) (domain.User, error) { return m.user, nil }
func (m *memoryStore) CreateSession(_ context.Context, sid, uid, hash string, expires time.Time) error {
	m.sessions[sid] = true
	m.tokens[hash] = &memoryToken{sessionID: sid, userID: uid, hash: hash, expires: expires}
	return nil
}
func (m *memoryStore) CreateSessionRefresh(_ context.Context, sid, uid, hash string, expires time.Time) error {
	m.tokens[hash] = &memoryToken{sessionID: sid, userID: uid, hash: hash, expires: expires}
	return nil
}
func (m *memoryStore) SessionActive(_ context.Context, sid, uid string) (bool, error) {
	return m.sessions[sid] && m.user.ID == uid, nil
}
func (m *memoryStore) RevokeSession(_ context.Context, sid string) error {
	m.sessions[sid] = false
	for _, token := range m.tokens {
		if token.sessionID == sid {
			token.revoked = true
		}
	}
	return nil
}
func (m *memoryStore) RotateRefresh(_ context.Context, hash string, now time.Time) (Rotation, error) {
	token := m.tokens[hash]
	if token == nil {
		return Rotation{}, domain.ErrUnauthorized
	}
	if token.used || token.revoked {
		m.sessions[token.sessionID] = false
		return Rotation{}, ErrTokenReused
	}
	if !token.expires.After(now) {
		return Rotation{}, domain.ErrUnauthorized
	}
	token.used = true
	return Rotation{User: m.user, SessionID: token.sessionID}, nil
}
func testService(t *testing.T) (*Service, *memoryStore) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := DefaultArgon2id().Hash("Owner123!")
	if err != nil {
		t.Fatal(err)
	}
	u := domain.User{ID: uuid.NewString(), Username: "owner", PasswordHash: hash, Roles: []string{"user"}}
	store := newMemoryStore(u)
	s := NewService(store, key, "test")
	s.hasher = Argon2id{Time: 1, Memory: 8192, Threads: 1, KeyLength: 16}
	s.now = time.Now
	return s, store
}
func TestPasswordArgon2id(t *testing.T) {
	hasher := Argon2id{Time: 1, Memory: 8192, Threads: 1, KeyLength: 16}
	hash, err := hasher.Hash("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !Verify("correct horse", hash) || Verify("wrong password", hash) {
		t.Fatal("password verification failed")
	}
	if Verify("correct horse", "not-a-hash") {
		t.Fatal("malformed hash accepted")
	}
}
func TestLoginAuthenticateRefreshRotationAndLogout(t *testing.T) {
	s, store := testService(t)
	tokens, err := s.Login(context.Background(), "owner", "Owner123!")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.ExpiresIn != 900 {
		t.Fatal("access TTL is not 15 minutes")
	}
	u, sid, err := s.Authenticate(context.Background(), tokens.AccessToken)
	if err != nil || u.ID != store.user.ID {
		t.Fatalf("authenticate: %v", err)
	}
	rotated, err := s.Refresh(context.Background(), tokens.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == tokens.RefreshToken {
		t.Fatal("refresh token not rotated")
	}
	if _, err := s.Refresh(context.Background(), tokens.RefreshToken); !errors.Is(err, ErrTokenReused) {
		t.Fatalf("reuse not detected: %v", err)
	}
	if _, _, err := s.Authenticate(context.Background(), rotated.AccessToken); err == nil {
		t.Fatal("refresh reuse did not revoke session")
	}
	_ = sid
	fresh, err := s.Login(context.Background(), "owner", "Owner123!")
	if err != nil {
		t.Fatal(err)
	}
	_, freshSID, _ := s.Authenticate(context.Background(), fresh.AccessToken)
	if err := s.Logout(context.Background(), freshSID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Authenticate(context.Background(), fresh.AccessToken); err == nil {
		t.Fatal("logged out access token accepted")
	}
}
func TestWrongPassword(t *testing.T) {
	s, _ := testService(t)
	if _, err := s.Login(context.Background(), "owner", "bad-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("wrong password accepted")
	}
}

package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrTokenReused = errors.New("refresh token reused")

type RefreshToken struct {
	ID, SessionID, UserID, TokenHash, ReplacedBy string
	ExpiresAt                                    time.Time
	Used, Revoked                                bool
}
type Rotation struct {
	User      domain.User
	SessionID string
	TokenID   string
}
type Store interface {
	UserByUsername(context.Context, string) (domain.User, error)
	UserByID(context.Context, string) (domain.User, error)
	CreateSession(context.Context, string, string, string, time.Time) error
	CreateSessionRefresh(context.Context, string, string, string, time.Time) error
	RotateRefresh(context.Context, string, time.Time) (Rotation, error)
	RevokeSession(context.Context, string) error
	SessionActive(context.Context, string, string) (bool, error)
}

type Claims struct {
	UserID string   `json:"uid"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}
type Service struct {
	store      Store
	key        *ecdsa.PrivateKey
	hasher     Argon2id
	issuer     string
	refreshTTL time.Duration
	now        func() time.Time
}
type Tokens struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int         `json:"expires_in"`
	User         domain.User `json:"user"`
}

func NewService(store Store, key *ecdsa.PrivateKey, issuer string) *Service {
	return &Service{store: store, key: key, hasher: DefaultArgon2id(), issuer: issuer, refreshTTL: 30 * 24 * time.Hour, now: time.Now}
}

func (s *Service) Login(ctx context.Context, username, password string) (Tokens, error) {
	user, err := s.store.UserByUsername(ctx, strings.TrimSpace(username))
	if err != nil || !Verify(password, user.PasswordHash) {
		return Tokens{}, ErrInvalidCredentials
	}
	return s.issue(ctx, user, "")
}

func (s *Service) issue(ctx context.Context, user domain.User, existingSession string) (Tokens, error) {
	now := s.now().UTC()
	sessionID := existingSession
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	refresh, hash, err := opaqueToken()
	if err != nil {
		return Tokens{}, err
	}
	claims := Claims{UserID: user.ID, Roles: user.Roles, RegisteredClaims: jwt.RegisteredClaims{ID: uuid.NewString(), Subject: user.ID, Issuer: s.issuer, IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute))}}
	// sid is deliberately registered as a private claim while keeping standard JWT fields.
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"iss": s.issuer, "sub": user.ID, "uid": user.ID, "sid": sessionID, "jti": claims.ID, "roles": user.Roles, "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(15 * time.Minute).Unix()})
	access, err := token.SignedString(s.key)
	if err != nil {
		return Tokens{}, err
	}
	if existingSession == "" {
		if err := s.store.CreateSession(ctx, sessionID, user.ID, hash, now.Add(s.refreshTTL)); err != nil {
			return Tokens{}, err
		}
	} else {
		if err := s.store.CreateSessionRefresh(ctx, sessionID, user.ID, hash, now.Add(s.refreshTTL)); err != nil {
			return Tokens{}, err
		}
	}
	return Tokens{access, refresh, "Bearer", 900, user}, nil
}

func (s *Service) Refresh(ctx context.Context, presented string) (Tokens, error) {
	rotation, err := s.store.RotateRefresh(ctx, HashToken(presented), s.now())
	if err != nil {
		return Tokens{}, err
	}
	tokens, err := s.issue(ctx, rotation.User, rotation.SessionID)
	if err == nil {
		return tokens, nil
	}
	return Tokens{}, err
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.store.RevokeSession(ctx, sessionID)
}

func (s *Service) Authenticate(ctx context.Context, raw string) (domain.User, string, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodES256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return &s.key.PublicKey, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithValidMethods([]string{"ES256"}))
	if err != nil || !token.Valid {
		return domain.User{}, "", domain.ErrUnauthorized
	}
	uid, _ := claims["uid"].(string)
	sid, _ := claims["sid"].(string)
	if uid == "" || sid == "" {
		return domain.User{}, "", domain.ErrUnauthorized
	}
	active, err := s.store.SessionActive(ctx, sid, uid)
	if err != nil || !active {
		return domain.User{}, "", domain.ErrUnauthorized
	}
	user, err := s.store.UserByID(ctx, uid)
	if err != nil {
		return domain.User{}, "", domain.ErrUnauthorized
	}
	return user, sid, nil
}

func opaqueToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	_, err = rand.Read(b)
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw), err
}
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

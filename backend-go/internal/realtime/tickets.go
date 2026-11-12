package realtime

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
)

type ticket struct {
	UserID    string
	ExpiresAt time.Time
}
type Tickets struct {
	mu    sync.Mutex
	items map[[32]byte]ticket
	ttl   time.Duration
	now   func() time.Time
}

func NewTickets(ttl time.Duration) *Tickets {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &Tickets{items: map[[32]byte]ticket{}, ttl: ttl, now: time.Now}
}
func (t *Tickets) Issue(userID string) (string, int64, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", 0, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	expires := t.now().Add(t.ttl)
	t.mu.Lock()
	defer t.mu.Unlock()
	for key, item := range t.items {
		if !item.ExpiresAt.After(t.now()) {
			delete(t.items, key)
		}
	}
	t.items[sum] = ticket{UserID: userID, ExpiresAt: expires}
	return token, expires.UnixMilli(), nil
}
func (t *Tickets) RedeemUserID(token string) (string, error) {
	sum := sha256.Sum256([]byte(token))
	t.mu.Lock()
	defer t.mu.Unlock()
	item, ok := t.items[sum]
	delete(t.items, sum)
	if !ok || !item.ExpiresAt.After(t.now()) {
		return "", domain.ErrUnauthorized
	}
	return item.UserID, nil
}

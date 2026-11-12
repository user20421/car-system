package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/car-system/car-system/backend-go/internal/domain"
)

type Event struct {
	Type         string `json:"type"`
	OccurredAtMS int64  `json:"occurred_at_ms"`
	Data         any    `json:"data"`
}
type Client struct {
	UserID  string
	Allowed map[string]bool
	Send    chan []byte
}
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

func NewHub() *Hub { return &Hub{clients: map[*Client]struct{}{}} }
func (h *Hub) Register(userID string, vehicles []domain.Vehicle) *Client {
	c := &Client{UserID: userID, Allowed: map[string]bool{}, Send: make(chan []byte, 64)}
	for _, v := range vehicles {
		c.Allowed[v.VIN] = true
	}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	return c
}
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.Send)
	}
	h.mu.Unlock()
}
func (h *Hub) broadcast(vin, kind string, data any, at int64) {
	raw, err := json.Marshal(Event{Type: kind, OccurredAtMS: at, Data: data})
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if !c.Allowed[vin] {
			continue
		}
		select {
		case c.Send <- raw:
		default:
		}
	}
}
func (h *Hub) Telemetry(t domain.Telemetry) { h.broadcast(t.VIN, "telemetry", t, t.OccurredAtMS) }
func (h *Hub) Command(c domain.Command)     { h.broadcast(c.VIN, "command", c, c.UpdatedAtMS) }
func (h *Hub) Alert(a domain.Alert)         { h.broadcast(a.VIN, "alert", a, time.Now().UTC().UnixMilli()) }

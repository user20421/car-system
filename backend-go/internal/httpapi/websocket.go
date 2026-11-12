package httpapi

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func (a *API) websocket(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	uid, err := a.Tickets.RedeemUserID(ticket)
	if err != nil {
		problem(w, r, 401, "INVALID_WS_TICKET", "Invalid or already used WebSocket ticket", "")
		return
	}
	u, err := a.Store.UserByID(r.Context(), uid)
	if err != nil {
		problem(w, r, 401, "INVALID_WS_TICKET", "Invalid WebSocket ticket", "")
		return
	}
	vehicles, _, err := a.Store.ListVehicles(r.Context(), u, 1, 10000)
	if err != nil {
		fail(w, r, err)
		return
	}
	a.upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		parsed, err := url.Parse(origin)
		return err == nil && strings.EqualFold(parsed.Host, r.Host)
	}
	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := a.Hub.Register(uid, vehicles)
	defer func() { a.Hub.Unregister(client); conn.Close() }()
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case raw, ok := <-client.Send:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		case <-r.Context().Done():
			return
		}
	}
}

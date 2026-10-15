package ws

import (
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"backend-go/pkg/logger"
)

// Hub 管理所有 WebSocket 连接，负责广播车况数据
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

var GlobalHub = NewHub()

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run 主循环：处理连接注册/注销/广播
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			logger.L.Info("WebSocket 客户端已连接", zap.Int("当前连接数", h.ClientCount()))
		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				_ = conn.Close()
			}
			h.mu.Unlock()
			logger.L.Info("WebSocket 客户端已断开", zap.Int("当前连接数", h.ClientCount()))
		case message := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					logger.L.Warn("WebSocket 推送失败", zap.Error(err))
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register 注册新连接
func (h *Hub) Register(conn *websocket.Conn) { h.register <- conn }

// Unregister 注销连接
func (h *Hub) Unregister(conn *websocket.Conn) { h.unregister <- conn }

// Broadcast 向所有客户端广播
func (h *Hub) Broadcast(msg []byte) { h.broadcast <- msg }

// ClientCount 当前连接数
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

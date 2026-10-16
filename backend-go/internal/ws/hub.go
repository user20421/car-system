package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"backend-go/pkg/logger"
)

const (
	sendBufferSize = 256              // 每客户端发送缓冲（约 4 分钟 1Hz 遥测）
	writeWait      = 5 * time.Second  // 单次写超时
	pingPeriod     = 30 * time.Second // 服务端心跳周期
)

// Client 单个 WebSocket 连接：独立发送缓冲 + 写协程，慢客户端不再阻塞全局广播
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	UserID   uint
	Username string
	Role     string
}

// NewClient 包装一个已升级的 WebSocket 连接
func NewClient(conn *websocket.Conn, userID uint, username, role string) *Client {
	return &Client{
		conn:     conn,
		send:     make(chan []byte, sendBufferSize),
		UserID:   userID,
		Username: username,
		Role:     role,
	}
}

// Hub 管理所有 WebSocket 连接，负责广播车况/指令/回执
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	done       chan struct{}
	stopOnce   sync.Once
}

var GlobalHub = NewHub()

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
	}
}

// Run 主循环：处理连接注册/注销/广播；收到 Stop 后关闭全部连接并退出
func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			h.mu.Lock()
			for c := range h.clients {
				close(c.send)
				if c.conn != nil {
					_ = c.conn.Close()
				}
				delete(h.clients, c)
			}
			h.mu.Unlock()
			logger.L.Info("WebSocket Hub 已停止")
			return
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			logger.L.Info("WebSocket 客户端已连接",
				zap.String("user", c.Username), zap.Int("当前连接数", h.ClientCount()))
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
				_ = c.conn.Close()
			}
			h.mu.Unlock()
			logger.L.Info("WebSocket 客户端已断开", zap.Int("当前连接数", h.ClientCount()))
		case message := <-h.broadcast:
			// 非阻塞分发：慢客户端丢帧，绝不拖累整体广播
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- message:
				default:
					logger.L.Debug("客户端发送缓冲已满，丢弃一帧", zap.String("user", c.Username))
				}
			}
			h.mu.RUnlock()
		}
	}
}

// WritePump 客户端独立写协程：从发送缓冲取消息写连接，附写超时与服务端心跳
func (c *Client) WritePump(h *Hub) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		h.Unregister(c)
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub 关闭：通知客户端后退出
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Register 注册新连接（Hub 已停止时直接关闭连接）
func (h *Hub) Register(c *Client) {
	select {
	case h.register <- c:
	case <-h.done:
		if c.conn != nil {
			_ = c.conn.Close()
		}
	}
}

// Unregister 注销连接（幂等；Hub 已停止时安全返回）
func (h *Hub) Unregister(c *Client) {
	select {
	case h.unregister <- c:
	case <-h.done:
	}
}

// Broadcast 向所有客户端广播（非阻塞：队列满则丢帧，不反压调用方）
func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		logger.L.Warn("广播队列已满，丢弃一帧")
	}
}

// ClientCount 当前连接数
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Stop 停止 Hub：关闭全部客户端连接并退出主循环（幂等）
func (h *Hub) Stop() {
	h.stopOnce.Do(func() { close(h.done) })
}

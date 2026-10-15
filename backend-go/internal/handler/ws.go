package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"backend-go/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 本地开发允许跨域
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocket 实时数据通道：前端/手机端连接后持续接收车况推送
func WebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	ws.GlobalHub.Register(conn)

	// 只接收不处理（车机数据走 HTTP 上报，这里纯下行推送）
	// 读循环用于感知客户端断开
	defer ws.GlobalHub.Unregister(conn)
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

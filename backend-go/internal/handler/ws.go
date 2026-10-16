package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"backend-go/internal/middleware"
	"backend-go/internal/ws"
	"backend-go/pkg/errs"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 本地开发允许跨域
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocket 实时数据通道：前端/手机端/车机连接后持续接收车况推送
// 鉴权：WS 握手无法携带 Authorization 头，统一约定 ?token=<jwt> 查询参数
func WebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		errs.Fail(c, http.StatusUnauthorized, errs.Unauthorized, "缺少 token 参数")
		return
	}
	claims, err := middleware.ParseToken(token)
	if err != nil {
		errs.Fail(c, http.StatusUnauthorized, errs.TokenInvalid, "Token 无效或已过期")
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := ws.NewClient(conn, claims.UserID, claims.Username, claims.Role)
	ws.GlobalHub.Register(client)
	go client.WritePump(ws.GlobalHub)

	// 只接收不处理（车机数据走 HTTP 上报，这里纯下行推送）
	// 读循环用于感知客户端断开
	defer ws.GlobalHub.Unregister(client)
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

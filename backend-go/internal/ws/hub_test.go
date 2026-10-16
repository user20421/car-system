package ws

import (
	"testing"
	"time"

	"backend-go/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init("release")
	m.Run()
}

// TestHubGracefulStop 验证 Hub 收到 Stop 后主循环退出且客户端被清理
func TestHubGracefulStop(t *testing.T) {
	h := NewHub()
	exited := make(chan struct{})
	go func() {
		h.Run()
		close(exited)
	}()

	h.Register(NewClient(nil, 1, "tester", "owner"))
	// 等待注册生效
	deadline := time.Now().Add(2 * time.Second)
	for h.ClientCount() != 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if h.ClientCount() != 1 {
		t.Fatalf("注册失败，连接数 = %d", h.ClientCount())
	}

	h.Broadcast([]byte(`{"type":"test"}`)) // 不应阻塞

	h.Stop()
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("Hub 未在超时内退出")
	}
	if n := h.ClientCount(); n != 0 {
		t.Fatalf("停止后仍有残留连接: %d", n)
	}

	// 停止后的注册/注销/广播必须安全返回（不阻塞、不 panic）
	h.Unregister(NewClient(nil, 2, "late", "owner"))
	h.Register(NewClient(nil, 3, "late", "owner"))
	h.Broadcast([]byte(`{"type":"after-stop"}`))
}

// TestBroadcastNonBlocking 广播通道满时不得阻塞调用方（反压防护）
func TestBroadcastNonBlocking(t *testing.T) {
	h := NewHub() // 不启动 Run，broadcast 通道无人消费
	payload := []byte("x")
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ { // 远超 256 缓冲
			h.Broadcast(payload)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Broadcast 发生阻塞（反压）")
	}
}

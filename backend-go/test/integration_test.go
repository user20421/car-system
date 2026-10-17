// Package test 集成测试：对真实中间件（MySQL/Redis/RabbitMQ）跑核心业务闭环。
// 中间件不可用时整体跳过（CI 中由 services 提供，本地由 docker compose 提供）。
//
//	go test ./test/ -v          # 跑集成测试
//	go test ./test/ -race -v    # 附并发竞争检测
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"backend-go/internal/cache"
	"backend-go/internal/config"
	"backend-go/internal/database"
	"backend-go/internal/middleware"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/internal/router"
	"backend-go/internal/service"
	"backend-go/internal/ws"
	"backend-go/pkg/logger"

	"golang.org/x/crypto/bcrypt"
)

const testVIN = "LVSFACBE8MF012345"

var (
	engine *gin.Engine
	cfg    *config.Config
)

// ---------------- 测试环境装配 ----------------

func TestMain(m *testing.M) {
	if !infraAvailable() {
		fmt.Println("SKIP: MySQL/Redis/RabbitMQ 不可用（请先 docker compose up -d）")
		os.Exit(0)
	}
	var err error
	cfg, err = config.Load("../config/config.yaml")
	if err != nil {
		panic(err)
	}
	logger.Init("release")
	middleware.InitJWT(cfg.JWT.Secret, cfg.JWT.ExpireDuration())

	must(database.Init(cfg.MySQL, "release"))
	must(cache.Init(cfg.Redis))
	must(mq.Init(cfg.RabbitMQ))

	ensureSeedData()
	service.SeedAlarmRules()

	// 启动后台组件：WS Hub、遥测消费者、Outbox 投递器
	ctx, cancel := context.WithCancel(context.Background())
	go ws.GlobalHub.Run()
	go func() { _ = mq.Client.ConsumeTelemetry(ctx, service.PersistTelemetry) }()
	go service.StartOutboxRelay(ctx)

	engine = router.Setup("release")

	// 注意：os.Exit 不执行 defer，必须显式清理
	code := m.Run()
	cancel()
	ws.GlobalHub.Stop()
	mq.Client.Close()
	cache.Close()
	database.Close()
	os.Exit(code)
}

// infraAvailable 快速探测三个中间件端口
func infraAvailable() bool {
	for _, addr := range []string{"127.0.0.1:3306", "127.0.0.1:6379", "127.0.0.1:5672"} {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return false
		}
		_ = conn.Close()
	}
	return true
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// ensureSeedData 保证 admin / driver1 / 示例车辆存在（与 main.seedData 一致）
func ensureSeedData() {
	var admin model.User
	if err := database.DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
		admin = model.User{Username: "admin", PasswordHash: string(hash), Role: "admin"}
		database.DB.Create(&admin)
	}
	var driver model.User
	if err := database.DB.Where("username = ?", "driver1").First(&driver).Error; err != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("driver123456"), bcrypt.DefaultCost)
		driver = model.User{Username: "driver1", PasswordHash: string(hash), Role: "owner"}
		database.DB.Create(&driver)
	}
	var v model.Vehicle
	if err := database.DB.Where("vin = ?", testVIN).First(&v).Error; err != nil {
		database.DB.Create(&model.Vehicle{VIN: testVIN, Model: "Test EV", OwnerID: driver.ID, Status: "online"})
	}
}

// ---------------- 测试工具 ----------------

// doRequest 发送 HTTP 请求并返回状态码与解析后的响应
func doRequest(t *testing.T, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

// login 登录并返回 token
func login(t *testing.T, username, password string) string {
	t.Helper()
	code, resp := doRequest(t, "POST", "/api/v1/auth/login", "", map[string]string{
		"username": username, "password": password,
	})
	if code != 200 {
		t.Fatalf("登录失败: status=%d resp=%v", code, resp)
	}
	return resp["data"].(map[string]any)["token"].(string)
}

// waitFor 轮询直到条件成立或超时
func waitFor(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("等待超时: %s", what)
}

func reportTelemetry(t *testing.T, token string, body map[string]any) {
	t.Helper()
	body["vin"] = testVIN
	code, resp := doRequest(t, "POST", "/api/v1/vehicle/telemetry", token, body)
	if code != 200 {
		t.Fatalf("车况上报失败: status=%d resp=%v", code, resp)
	}
}

// ---------------- 集成测试用例 ----------------

// TestLoginAndProfile 登录 → 携带 token 访问受保护接口
func TestLoginAndProfile(t *testing.T) {
	token := login(t, "admin", "admin123456")
	code, _ := doRequest(t, "GET", "/api/v1/profile", token, nil)
	if code != 200 {
		t.Fatalf("profile 访问失败: %d", code)
	}

	// 错误密码应 401
	code, _ = doRequest(t, "POST", "/api/v1/auth/login", "", map[string]string{
		"username": "admin", "password": "wrong-password",
	})
	if code == 200 {
		t.Fatal("错误密码不应登录成功")
	}
}

// TestTelemetryFlow 上报车况 → 最新车况可查（Redis）→ 异步落库（MQ → MySQL）
func TestTelemetryFlow(t *testing.T) {
	token := login(t, "admin", "admin123456")
	reportTelemetry(t, token, map[string]any{
		"speed": 66.6, "rpm": 2800, "battery": 88.8, "lat": 31.2301, "lng": 121.4701,
	})

	// 最新车况立即可查（Redis 实时缓存）
	code, resp := doRequest(t, "GET", "/api/v1/vehicle/"+testVIN+"/telemetry/latest", token, nil)
	if code != 200 {
		t.Fatalf("查询最新车况失败: %d", code)
	}
	data := resp["data"].(map[string]any)
	if data["speed"].(float64) != 66.6 {
		t.Fatalf("最新车况不符: %v", data["speed"])
	}

	// 异步落库：MQ 消费者最终写入 MySQL（按增量断言，避免历史残留假阳性）
	var before int64
	database.DB.Model(&model.Telemetry{}).Where("speed = ? AND battery = ?", 66.6, 88.8).Count(&before)
	waitFor(t, "遥测异步落库", 8*time.Second, func() bool {
		var count int64
		database.DB.Model(&model.Telemetry{}).
			Where("speed = ? AND battery = ?", 66.6, 88.8).Count(&count)
		return count > before
	})
}

// TestTelemetryIdempotency 同一 message_id 重复投递只落库一次
func TestTelemetryIdempotency(t *testing.T) {
	msgID := fmt.Sprintf("IDEMPOTENT-IT-%d", time.Now().UnixNano())
	body, _ := json.Marshal(map[string]any{
		"vin": testVIN, "message_id": msgID,
		"speed": 1.1, "rpm": 100, "battery": 50, "lat": 31.2, "lng": 121.4,
	})
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := mq.Client.PublishTelemetry(ctx, testVIN, body); err != nil {
			t.Fatalf("发布失败: %v", err)
		}
	}
	waitFor(t, "重复消息处理", 8*time.Second, func() bool {
		var count int64
		database.DB.Model(&model.Telemetry{}).Where("message_id = ?", msgID).Count(&count)
		return count >= 1
	})
	var count int64
	database.DB.Model(&model.Telemetry{}).Where("message_id = ?", msgID).Count(&count)
	if count != 1 {
		t.Fatalf("幂等失败：同一消息落库 %d 次（应为 1）", count)
	}
}

// TestCommandFlow 下发指令 → Outbox 投递 → 状态机流转
func TestCommandFlow(t *testing.T) {
	token := login(t, "admin", "admin123456")
	code, resp := doRequest(t, "POST", "/api/v1/vehicle/"+testVIN+"/command", token,
		map[string]string{"command": "honk"})
	if code != 200 {
		t.Fatalf("下发指令失败: %d %v", code, resp)
	}
	cmdID := uint(resp["data"].(map[string]any)["id"].(float64))

	// Outbox 投递器应将其投递到 MQ（状态 sent）
	waitFor(t, "Outbox 投递", 8*time.Second, func() bool {
		var out model.OutboxMessage
		if err := database.DB.Where("routing_key = ?", "vehicle.control."+testVIN).
			Order("id DESC").First(&out).Error; err != nil {
			return false
		}
		return out.Status == "sent"
	})

	// 非法指令应被拒绝
	code, _ = doRequest(t, "POST", "/api/v1/vehicle/"+testVIN+"/command", token,
		map[string]string{"command": "explode"})
	if code == 200 {
		t.Fatal("非法指令不应被接受")
	}

	// 指令历史可查
	code, resp = doRequest(t, "GET", "/api/v1/vehicle/"+testVIN+"/commands", token, nil)
	if code != 200 || resp["data"] == nil {
		t.Fatalf("指令历史查询失败: %d", code)
	}
	_ = cmdID
}

// TestOwnershipEnforced 越权防护：无关用户不能查看/控制他人车辆
func TestOwnershipEnforced(t *testing.T) {
	// 临时用户
	hash, _ := bcrypt.GenerateFromPassword([]byte("test123456"), bcrypt.DefaultCost)
	tmp := model.User{Username: "it_temp_user", PasswordHash: string(hash), Role: "owner"}
	database.DB.Where("username = ?", tmp.Username).Delete(&model.User{})
	must(database.DB.Create(&tmp).Error)
	defer database.DB.Delete(&tmp)

	token := login(t, "it_temp_user", "test123456")

	if code, _ := doRequest(t, "GET", "/api/v1/vehicle/"+testVIN, token, nil); code != 403 {
		t.Fatalf("越权查看应返回 403，实际 %d", code)
	}
	if code, _ := doRequest(t, "POST", "/api/v1/vehicle/"+testVIN+"/command", token,
		map[string]string{"command": "unlock"}); code != 403 {
		t.Fatalf("越权控车应返回 403，实际 %d", code)
	}

	// 车主本人可以访问
	ownerToken := login(t, "driver1", "driver123456")
	if code, _ := doRequest(t, "GET", "/api/v1/vehicle/"+testVIN, ownerToken, nil); code != 200 {
		t.Fatalf("车主访问自己的车应返回 200，实际 %d", code)
	}
}

// TestWSRequiresAuth WebSocket 未带 token 应被拒绝
func TestWSRequiresAuth(t *testing.T) {
	code, _ := doRequest(t, "GET", "/ws", "", nil)
	if code != 401 {
		t.Fatalf("WS 无 token 应返回 401，实际 %d", code)
	}
}

// TestAlarmLifecycle 自建专用规则：触发 → 恢复（不依赖种子数据，跑完即清理）
func TestAlarmLifecycle(t *testing.T) {
	token := login(t, "admin", "admin123456")

	// 本用例专用规则：battery < 5（种子规则阈值 20，互不干扰）
	rule := model.AlarmRule{Name: "测试极低电量", Metric: "battery", Operator: "lt",
		Threshold: 5, Severity: "info", Enabled: true}
	must(database.DB.Create(&rule).Error)
	service.SeedAlarmRules() // 无害调用
	service.InvalidateAlarmCache()
	defer func() {
		database.DB.Delete(&rule)
		database.DB.Where("rule_id = ?", rule.ID).Delete(&model.AlarmRecord{})
		service.InvalidateAlarmCache()
	}()

	// 触发：电量 3 < 阈值 5
	reportTelemetry(t, token, map[string]any{
		"speed": 30, "rpm": 1500, "battery": 3, "lat": 31.2301, "lng": 121.4701,
	})
	waitFor(t, "低电量告警触发", 8*time.Second, func() bool {
		var count int64
		database.DB.Model(&model.AlarmRecord{}).
			Where("rule_id = ? AND vin = ? AND status = ?", rule.ID, testVIN, "active").Count(&count)
		return count > 0
	})

	// 恢复：电量回到 90
	reportTelemetry(t, token, map[string]any{
		"speed": 30, "rpm": 1500, "battery": 90, "lat": 31.2301, "lng": 121.4701,
	})
	waitFor(t, "低电量告警恢复", 8*time.Second, func() bool {
		var count int64
		database.DB.Model(&model.AlarmRecord{}).
			Where("rule_id = ? AND vin = ? AND status = ?", rule.ID, testVIN, "active").Count(&count)
		return count == 0
	})
}

package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"

	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/internal/service"
	"backend-go/pkg/logger"
)

// Params 虚拟车队运行参数（可热调节）
type Params struct {
	VehicleCount int     `json:"vehicle_count"` // 1 ~ 50
	IntervalMs   int     `json:"interval_ms"`   // 100 ~ 5000
	FaultRate    float64 `json:"fault_rate"`    // 0 ~ 0.1，每帧触发故障的概率
}

// vehicleState 单辆虚拟车的运行时状态
type vehicleState struct {
	vin            string
	tick           int
	battery        float64
	lat, lng       float64
	faultCountdown int
	currentFault   string
}

// Fleet 虚拟车队引擎
type Fleet struct {
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	params  Params
}

// Global 全局虚拟车队实例
var Global = &Fleet{}

// Start 启动虚拟车队（已运行则先停再以新参数重启）
func (f *Fleet) Start(p Params) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.running {
		f.cancel()
		f.running = false
	}

	// 虚拟车辆挂到 admin 名下（满足外键约束）
	var admin model.User
	if err := database.DB.Where("role = ?", "admin").First(&admin).Error; err != nil {
		return fmt.Errorf("未找到管理员用户: %w", err)
	}

	// 确保虚拟车辆已登记入库
	states := make([]*vehicleState, 0, p.VehicleCount)
	for i := 0; i < p.VehicleCount; i++ {
		vin := fmt.Sprintf("SIM%014d", i+1) // 17 位模拟 VIN
		var v model.Vehicle
		if err := database.DB.Where("vin = ?", vin).First(&v).Error; err != nil {
			v = model.Vehicle{
				VIN:     vin,
				Model:   fmt.Sprintf("Virtual EV #%d", i+1),
				OwnerID: admin.ID,
				Status:  "online",
			}
			if err := database.DB.Create(&v).Error; err != nil {
				return fmt.Errorf("登记虚拟车辆失败: %w", err)
			}
		}
		states = append(states, &vehicleState{
			vin:     vin,
			battery: 60 + rand.Float64()*40,
			lat:     31.23 + (rand.Float64()-0.5)*0.1, // 上海周边散布
			lng:     121.47 + (rand.Float64()-0.5)*0.1,
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.params = p
	f.running = true

	for _, st := range states {
		go f.runVehicle(ctx, st)
	}

	// 虚拟车控车指令消费者：按每辆车的 VIN 精确绑定，模拟执行后自动回执
	bindingKeys := make([]string, 0, len(states))
	for _, st := range states {
		bindingKeys = append(bindingKeys, "vehicle.control."+st.vin)
	}
	go func() {
		err := mq.Client.ConsumeControl(ctx, "", bindingKeys, "sim-fleet-executor",
			func(body []byte) error {
				var msg service.ControlMessage
				if err := json.Unmarshal(body, &msg); err != nil {
					return err
				}
				// 模拟车机执行耗时 1~2 秒后回执
				time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)
				return service.AckCommand(msg.VIN, msg.ID, "executed")
			})
		if err != nil {
			logger.L.Warn("虚拟车队控车消费者退出", zap.Error(err))
		}
	}()
	logger.L.Info("虚拟车队已启动",
		zap.Int("vehicles", p.VehicleCount),
		zap.Int("interval_ms", p.IntervalMs),
		zap.Float64("fault_rate", p.FaultRate))
	return nil
}

// Stop 停止虚拟车队
func (f *Fleet) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.running {
		f.cancel()
		f.running = false
		logger.L.Info("虚拟车队已停止")
	}
}

// Status 当前状态
func (f *Fleet) Status() (bool, Params) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running, f.params
}

// runVehicle 单辆车的数据产生循环
func (f *Fleet) runVehicle(ctx context.Context, st *vehicleState) {
	ticker := time.NewTicker(time.Duration(f.params.IntervalMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st.tick++
			in := f.generate(st)
			if err := service.Ingest(in); err != nil {
				logger.L.Warn("虚拟车上报失败", zap.String("vin", st.vin), zap.Error(err))
			}
		}
	}
}

// generate 生成一帧模拟车况（与 Qt 端模拟器同策略）
func (f *Fleet) generate(st *vehicleState) service.TelemetryInput {
	// 车速：巡航正弦 + 噪声，周期性停车
	phase := float64(st.tick) / 120.0 * 2 * math.Pi
	base := 60.0 * (1.0 + math.Sin(phase))
	if math.Sin(phase*0.5) < -0.7 {
		base = 0
	}
	speed := math.Max(0, base+(rand.Float64()-0.5)*10)

	rpm := 800 + rand.Intn(100)
	if speed >= 1 {
		rpm = int(1200 + speed*25 + float64(rand.Intn(400)-200))
	}

	// 电量：行驶耗电 / 停车回充
	if speed < 1 {
		st.battery = math.Min(100, st.battery+0.05)
	} else {
		st.battery = math.Max(15, st.battery-0.02)
	}

	// GPS 漂移
	st.lat += (rand.Float64() - 0.5) * 0.0004
	st.lng += (rand.Float64() - 0.5) * 0.0004

	// 故障码：按故障率概率触发，持续 5~15 帧
	if st.faultCountdown > 0 {
		st.faultCountdown--
		if st.faultCountdown == 0 {
			st.currentFault = ""
		}
	} else if rand.Float64() < f.params.FaultRate {
		codes := []string{"P0300", "P0420", "B1001", "U0100"}
		st.currentFault = codes[rand.Intn(len(codes))]
		st.faultCountdown = 5 + rand.Intn(10)
	}

	return service.TelemetryInput{
		VIN:       st.vin,
		Speed:     math.Round(speed*10) / 10,
		RPM:       rpm,
		Battery:   math.Round(st.battery*10) / 10,
		Lat:       st.lat,
		Lng:       st.lng,
		FaultCode: st.currentFault,
	}
}

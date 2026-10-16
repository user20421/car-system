package service

// 告警引擎：遥测数据到达时同步评估告警规则与电子围栏。
// 生命周期：条件成立且无活动告警 → 触发（active）；条件不再成立且有活动告警 → 恢复（resolved）。
// 规则/围栏缓存在内存（10s TTL + 变更时主动失效），避免每帧遥测都打数据库。

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"

	"backend-go/internal/cache"
	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/internal/ws"
	"backend-go/pkg/logger"
)

const (
	ruleCacheTTL  = 10 * time.Second
	fenceCacheTTL = 10 * time.Second
	activeKeyTTL  = time.Hour // 告警/围栏状态在 Redis 的保持时间（每帧续期）
)

var (
	rulesCache     []model.AlarmRule
	rulesLoadedAt  time.Time
	rulesMu        sync.RWMutex
	fencesCache    []model.GeoFence
	fencesLoadedAt time.Time
	fencesMu       sync.RWMutex
)

// InvalidateAlarmCache 规则/围栏变更后调用，下次评估立即重新加载
func InvalidateAlarmCache() {
	rulesMu.Lock()
	rulesLoadedAt = time.Time{}
	rulesMu.Unlock()
	fencesMu.Lock()
	fencesLoadedAt = time.Time{}
	fencesMu.Unlock()
}

// SeedAlarmRules 初始化默认告警规则（幂等）
func SeedAlarmRules() {
	var count int64
	database.DB.Model(&model.AlarmRule{}).Count(&count)
	if count > 0 {
		return
	}
	defaults := []model.AlarmRule{
		{Name: "电量过低", Metric: "battery", Operator: "lt", Threshold: 20, Severity: "warning"},
		{Name: "故障码告警", Metric: "fault_code", Operator: "ne", Threshold: 0, Severity: "critical"},
		{Name: "严重超速", Metric: "speed", Operator: "gt", Threshold: 120, Severity: "warning"},
	}
	if err := database.DB.Create(&defaults).Error; err != nil {
		logger.L.Warn("默认告警规则初始化失败", zap.Error(err))
		return
	}
	logger.L.Info("已创建默认告警规则", zap.Int("count", len(defaults)))
}

// EvaluateAlarms 遥测帧的告警评估入口（规则判定 + 围栏进出）
func EvaluateAlarms(in TelemetryInput, vehicle model.Vehicle) {
	evaluateRules(in, vehicle)
	evaluateFences(in, vehicle)
}

// ---------------- 规则判定 ----------------

func loadRules() []model.AlarmRule {
	rulesMu.RLock()
	if time.Since(rulesLoadedAt) < ruleCacheTTL {
		defer rulesMu.RUnlock()
		return rulesCache
	}
	rulesMu.RUnlock()

	rulesMu.Lock()
	defer rulesMu.Unlock()
	if time.Since(rulesLoadedAt) < ruleCacheTTL { // double-check
		return rulesCache
	}
	var rules []model.AlarmRule
	if err := database.DB.Where("enabled = ?", true).Find(&rules).Error; err != nil {
		logger.L.Warn("加载告警规则失败", zap.Error(err))
		return rulesCache
	}
	rulesCache = rules
	rulesLoadedAt = time.Now()
	return rulesCache
}

func evaluateRules(in TelemetryInput, vehicle model.Vehicle) {
	for _, rule := range loadRules() {
		triggered := ruleTriggered(rule, in)
		activeKey := fmt.Sprintf("alarm:active:rule:%d:%s", rule.ID, in.VIN)
		if triggered {
			triggerAlarm(activeKey, model.AlarmRecord{
				RuleID:    rule.ID,
				VehicleID: vehicle.ID,
				VIN:       in.VIN,
				Title:     rule.Name,
				Detail:    ruleDetail(rule, in),
				Severity:  rule.Severity,
			})
		} else {
			resolveAlarm(activeKey, rule.ID, 0, in.VIN)
		}
	}
}

// ruleTriggered 判断单条规则是否命中
func ruleTriggered(rule model.AlarmRule, in TelemetryInput) bool {
	var v float64
	switch rule.Metric {
	case "speed":
		v = in.Speed
	case "rpm":
		v = float64(in.RPM)
	case "battery":
		v = in.Battery
	case "fault_code":
		return in.FaultCode != "" // 有故障码即命中
	default:
		return false
	}
	switch rule.Operator {
	case "lt":
		return v < rule.Threshold
	case "lte":
		return v <= rule.Threshold
	case "gt":
		return v > rule.Threshold
	case "gte":
		return v >= rule.Threshold
	}
	return false
}

func ruleDetail(rule model.AlarmRule, in TelemetryInput) string {
	if rule.Metric == "fault_code" {
		return fmt.Sprintf("车辆 %s 上报故障码 %s", in.VIN, in.FaultCode)
	}
	var v float64
	switch rule.Metric {
	case "speed":
		v = in.Speed
	case "rpm":
		v = float64(in.RPM)
	case "battery":
		v = in.Battery
	}
	return fmt.Sprintf("车辆 %s 当前 %s=%.1f，阈值 %s %.1f", in.VIN, rule.Metric, v, rule.Operator, rule.Threshold)
}

// ---------------- 围栏判定 ----------------

func loadFences() []model.GeoFence {
	fencesMu.RLock()
	if time.Since(fencesLoadedAt) < fenceCacheTTL {
		defer fencesMu.RUnlock()
		return fencesCache
	}
	fencesMu.RUnlock()

	fencesMu.Lock()
	defer fencesMu.Unlock()
	if time.Since(fencesLoadedAt) < fenceCacheTTL {
		return fencesCache
	}
	var fences []model.GeoFence
	if err := database.DB.Where("enabled = ?", true).Find(&fences).Error; err != nil {
		logger.L.Warn("加载电子围栏失败", zap.Error(err))
		return fencesCache
	}
	fencesCache = fences
	fencesLoadedAt = time.Now()
	return fencesCache
}

func evaluateFences(in TelemetryInput, vehicle model.Vehicle) {
	ctx := context.Background()
	for _, fence := range loadFences() {
		if fence.VIN != "" && fence.VIN != in.VIN {
			continue
		}
		inside := haversineMeters(fence.Lat, fence.Lng, in.Lat, in.Lng) <= fence.RadiusM
		stateKey := fmt.Sprintf("fence:state:%d:%s", fence.ID, in.VIN)

		prev, err := cache.RDB.Get(ctx, stateKey).Result()
		if err != nil {
			// 无历史状态（首次观测/重启后）：静默记录当前状态，不产生告警风暴
			setFenceState(ctx, stateKey, inside)
			continue
		}
		wasInside := prev == "1"
		if wasInside == inside {
			setFenceState(ctx, stateKey, inside) // 续期
			continue
		}
		// 状态翻转：进出边界
		setFenceState(ctx, stateKey, inside)
		entered := inside
		if fence.AlarmOn == "both" ||
			(entered && fence.AlarmOn == "enter") ||
			(!entered && fence.AlarmOn == "exit") {
			dir := "驶出"
			if entered {
				dir = "驶入"
			}
			triggerAlarm(fmt.Sprintf("alarm:active:fence:%d:%s", fence.ID, in.VIN), model.AlarmRecord{
				FenceID:   fence.ID,
				VehicleID: vehicle.ID,
				VIN:       in.VIN,
				Title:     fmt.Sprintf("围栏告警：%s", fence.Name),
				Detail:    fmt.Sprintf("车辆 %s %s围栏「%s」（圆心 %.4f,%.4f 半径 %.0fm）", in.VIN, dir, fence.Name, fence.Lat, fence.Lng, fence.RadiusM),
				Severity:  "warning",
			})
		}
	}
}

func setFenceState(ctx context.Context, key string, inside bool) {
	v := "0"
	if inside {
		v = "1"
	}
	_ = cache.RDB.Set(ctx, key, v, activeKeyTTL).Err()
}

// haversineMeters 两 GPS 坐标间的球面距离（米）
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadius * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// ---------------- 触发 / 恢复 ----------------

// triggerAlarm 触发告警：已有活动告警则仅续期防抖，否则落库 + WS 广播
func triggerAlarm(activeKey string, rec model.AlarmRecord) {
	ctx := context.Background()
	ok, err := cache.RDB.SetNX(ctx, activeKey, "1", activeKeyTTL).Result()
	if err != nil {
		logger.L.Warn("告警状态写入失败", zap.Error(err))
		return
	}
	if !ok {
		_ = cache.RDB.Expire(ctx, activeKey, activeKeyTTL) // 活动告警持续期间续期
		return
	}
	rec.Status = "active"
	rec.TriggeredAt = time.Now()
	if err := database.DB.Create(&rec).Error; err != nil {
		logger.L.Warn("告警落库失败", zap.Error(err))
		_ = cache.RDB.Del(ctx, activeKey)
		return
	}
	logger.L.Warn("告警触发", zap.String("title", rec.Title), zap.String("vin", rec.VIN))
	push, _ := json.Marshal(map[string]any{"type": "alarm", "data": rec})
	ws.GlobalHub.Broadcast(push)
}

// resolveAlarm 条件不再成立时恢复活动告警（按规则/围栏 + VIN 精确匹配）
func resolveAlarm(activeKey string, ruleID, fenceID uint, vin string) {
	ctx := context.Background()
	existed := cache.RDB.Del(ctx, activeKey).Val()
	if existed == 0 {
		return // 无活动告警
	}
	q := database.DB.Where("status = ? AND vin = ?", "active", vin)
	if ruleID > 0 {
		q = q.Where("rule_id = ?", ruleID)
	}
	if fenceID > 0 {
		q = q.Where("fence_id = ?", fenceID)
	}
	var rec model.AlarmRecord
	if err := q.Order("id DESC").First(&rec).Error; err != nil {
		return
	}
	now := time.Now()
	database.DB.Model(&rec).Updates(map[string]any{"status": "resolved", "resolved_at": &now})
	push, _ := json.Marshal(map[string]any{"type": "alarm_resolved", "data": rec})
	ws.GlobalHub.Broadcast(push)
}

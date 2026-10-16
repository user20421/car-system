package model

import "time"

// User 系统用户（管理员/车主）
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:128;not null" json:"-"`
	Role         string    `gorm:"size:20;not null;default:owner" json:"role"` // admin | owner
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Vehicle 车辆档案
type Vehicle struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VIN       string    `gorm:"uniqueIndex;size:17;not null" json:"vin"` // 车架号
	Model     string    `gorm:"size:64" json:"model"`
	OwnerID   uint      `gorm:"index" json:"owner_id"`
	Owner     User      `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Status    string    `gorm:"size:20;default:offline" json:"status"` // online | offline
	Locked    bool      `gorm:"default:true" json:"locked"`            // 车门锁状态
	AcOn      bool      `gorm:"default:false" json:"ac_on"`            // 空调状态
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Command 远程控车指令
type Command struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	VehicleID  uint       `gorm:"index" json:"vehicle_id"`
	Command    string     `gorm:"size:20;not null" json:"command"`              // lock/unlock/ac_on/ac_off/honk
	Status     string     `gorm:"size:20;not null;default:pending" json:"status"` // pending/sent/executed/failed
	Result     string     `gorm:"size:64" json:"result"`
	CreatedAt  time.Time  `json:"created_at"`
	ExecutedAt *time.Time `json:"executed_at"`
}

// Telemetry 车况遥测数据（高频写入）
type Telemetry struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VehicleID uint      `gorm:"index:idx_vehicle_time,priority:1" json:"vehicle_id"`
	MessageID string    `gorm:"uniqueIndex;size:64" json:"message_id"`   // 消息唯一标识，用于幂等去重
	Speed     float64   `json:"speed"`       // 车速 km/h
	RPM       int       `json:"rpm"`         // 发动机转速
	Battery   float64   `json:"battery"`     // 电量百分比
	Lat       float64   `json:"lat"`         // 纬度
	Lng       float64   `json:"lng"`         // 经度
	FaultCode string    `gorm:"size:32" json:"fault_code"` // 故障码，空为无故障
	CreatedAt time.Time `gorm:"index:idx_vehicle_time,priority:2" json:"created_at"`
}

// OutboxMessage 发件箱：业务数据与待投递消息在同一事务中写入，relay 异步投递到 MQ
// 解决"先落库后发 MQ，MQ 失败导致指令永久 pending"的双写不一致问题
type OutboxMessage struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Exchange    string     `gorm:"size:64" json:"exchange"`
	RoutingKey  string     `gorm:"size:128" json:"routing_key"`
	Body        string     `gorm:"type:text" json:"body"`
	TraceParent string     `gorm:"size:64" json:"-"`                                        // 链路追踪上下文，relay 投递时续链
	Status      string     `gorm:"size:20;not null;default:pending;index" json:"status"` // pending | sent | failed
	Retries     int        `json:"retries"`
	CreatedAt   time.Time  `json:"created_at"`
	SentAt      *time.Time `json:"sent_at"`
}

// DeadLetter 死信记录：消费重试耗尽后的消息快照，供后台查询与人工重投
type DeadLetter struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Queue      string    `gorm:"size:64;index" json:"queue"`
	Exchange   string    `gorm:"size:64" json:"exchange"`
	RoutingKey string    `gorm:"size:128" json:"routing_key"`
	Body       string    `gorm:"type:text" json:"body"`
	Reason     string    `gorm:"size:255" json:"reason"` // 最终失败原因
	CreatedAt  time.Time `json:"created_at"`
}

// AlarmRule 告警规则：对遥测指标做实时阈值判定
type AlarmRule struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Metric    string    `gorm:"size:20;not null" json:"metric"`                 // speed | rpm | battery | fault_code
	Operator  string    `gorm:"size:8;not null" json:"operator"`               // lt | lte | gt | gte | ne（fault_code 用 ne，threshold 固定 0）
	Threshold float64   `json:"threshold"`
	Severity  string    `gorm:"size:20;not null;default:warning" json:"severity"` // info | warning | critical
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AlarmRecord 告警记录：active → resolved 生命周期
type AlarmRecord struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	RuleID      uint       `gorm:"index" json:"rule_id"`  // >0 表示规则告警
	FenceID     uint       `gorm:"index" json:"fence_id"` // >0 表示围栏告警
	VehicleID   uint       `gorm:"index" json:"vehicle_id"`
	VIN         string     `gorm:"size:17;index" json:"vin"`
	Title       string     `gorm:"size:128" json:"title"`
	Detail      string     `gorm:"size:255" json:"detail"`
	Severity    string     `gorm:"size:20" json:"severity"`
	Status      string     `gorm:"size:20;not null;default:active;index" json:"status"` // active | resolved
	TriggeredAt time.Time  `json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// GeoFence 电子围栏（圆形）：车辆进出边界触发告警
type GeoFence struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	VIN       string    `gorm:"size:17;index" json:"vin"` // 空 = 对全部车辆生效
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	RadiusM   float64   `json:"radius_m"`                                 // 半径（米）
	AlarmOn   string    `gorm:"size:10;not null;default:exit" json:"alarm_on"` // enter | exit | both
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

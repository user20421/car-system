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
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Telemetry 车况遥测数据（高频写入）
type Telemetry struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	VehicleID uint      `gorm:"index:idx_vehicle_time,priority:1" json:"vehicle_id"`
	Speed     float64   `json:"speed"`       // 车速 km/h
	RPM       int       `json:"rpm"`         // 发动机转速
	Battery   float64   `json:"battery"`     // 电量百分比
	Lat       float64   `json:"lat"`         // 纬度
	Lng       float64   `json:"lng"`         // 经度
	FaultCode string    `gorm:"size:32" json:"fault_code"` // 故障码，空为无故障
	CreatedAt time.Time `gorm:"index:idx_vehicle_time,priority:2" json:"created_at"`
}

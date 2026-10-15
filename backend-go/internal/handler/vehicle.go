package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"backend-go/internal/cache"
	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/internal/service"
)

// TelemetryRequest 车机上报的车况数据
type TelemetryRequest struct {
	VIN       string  `json:"vin" binding:"required,len=17"`
	Speed     float64 `json:"speed" binding:"gte=0"`
	RPM       int     `json:"rpm" binding:"gte=0"`
	Battery   float64 `json:"battery" binding:"gte=0,lte=100"`
	Lat       float64 `json:"lat" binding:"gte=-90,lte=90"`
	Lng       float64 `json:"lng" binding:"gte=-180,lte=180"`
	FaultCode string  `json:"fault_code"`
}

// ReportTelemetry 车机端上报车况：写 Redis 实时状态 + RabbitMQ 异步落库 + WebSocket 广播
// @Summary 车机上报车况
// @Tags 车辆
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body TelemetryRequest true "车况数据"
// @Success 200 {object} map[string]any "上报成功"
// @Failure 400 {object} map[string]any "参数不合法"
// @Failure 404 {object} map[string]any "车辆不存在"
// @Router /api/v1/vehicle/telemetry [post]
func ReportTelemetry(c *gin.Context) {
	var req TelemetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 40001, "msg": "参数不合法: " + err.Error()})
		return
	}

	if err := service.Ingest(service.TelemetryInput(req)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40401, "msg": "车辆不存在，请先登记"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "上报成功"})
}

// GetLatestTelemetry 查询车辆最新状态（优先 Redis，回源 MySQL）
// @Summary 查询最新车况
// @Tags 车辆
// @Produce json
// @Security BearerAuth
// @Param vin path string true "车架号 (17位)"
// @Success 200 {object} map[string]any "data 为最新车况"
// @Failure 404 {object} map[string]any "车辆或数据不存在"
// @Router /api/v1/vehicle/{vin}/telemetry/latest [get]
func GetLatestTelemetry(c *gin.Context) {
	vin := c.Param("vin")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 优先读缓存
	if data, err := cache.RDB.Get(ctx, "vehicle:latest:"+vin).Result(); err == nil {
		var t TelemetryRequest
		_ = json.Unmarshal([]byte(data), &t)
		c.JSON(http.StatusOK, gin.H{"code": 0, "source": "redis", "data": t})
		return
	}

	// 缓存未命中，回源数据库
	var vehicle model.Vehicle
	if err := database.DB.Where("vin = ?", vin).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40401, "msg": "车辆不存在"})
		return
	}
	var t model.Telemetry
	if err := database.DB.Where("vehicle_id = ?", vehicle.ID).Order("created_at DESC").First(&t).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 40402, "msg": "暂无车况数据"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "source": "mysql", "data": t})
}

// ListVehicles 车辆列表（管理员）
// @Summary 车辆列表
// @Tags 车辆
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]any "data 为车辆数组"
// @Failure 403 {object} map[string]any "权限不足"
// @Router /api/v1/vehicle [get]
func ListVehicles(c *gin.Context) {
	var vehicles []model.Vehicle
	if err := database.DB.Preload("Owner").Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50005, "msg": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": vehicles, "total": len(vehicles)})
}

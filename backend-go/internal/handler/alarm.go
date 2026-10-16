package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/internal/service"
	"backend-go/pkg/errs"
)

// ---------------- 告警规则 CRUD ----------------

// ListAlarmRules 告警规则列表
// @Summary 告警规则列表
// @Tags 告警
// @Security BearerAuth
// @Success 200 {object} map[string]any "data 为规则数组"
// @Router /api/v1/alarm/rules [get]
func ListAlarmRules(c *gin.Context) {
	var rules []model.AlarmRule
	database.DB.Order("id").Find(&rules)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules})
}

// AlarmRuleRequest 规则创建/更新参数
type AlarmRuleRequest struct {
	Name      string  `json:"name" binding:"required,max=64"`
	Metric    string  `json:"metric" binding:"required,oneof=speed rpm battery fault_code"`
	Operator  string  `json:"operator" binding:"required,oneof=lt lte gt gte ne"`
	Threshold float64 `json:"threshold"`
	Severity  string  `json:"severity" binding:"required,oneof=info warning critical"`
	Enabled   *bool   `json:"enabled"`
}

// CreateAlarmRule 新建告警规则
// @Summary 新建告警规则
// @Tags 告警
// @Security BearerAuth
// @Param body body AlarmRuleRequest true "规则"
// @Success 200 {object} map[string]any "创建成功"
// @Router /api/v1/alarm/rules [post]
func CreateAlarmRule(c *gin.Context) {
	var req AlarmRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法")
		return
	}
	rule := model.AlarmRule{
		Name: req.Name, Metric: req.Metric, Operator: req.Operator,
		Threshold: req.Threshold, Severity: req.Severity, Enabled: true,
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if err := database.DB.Create(&rule).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "创建失败")
		return
	}
	service.InvalidateAlarmCache()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

// UpdateAlarmRule 更新告警规则（含启停）
// @Summary 更新告警规则
// @Tags 告警
// @Security BearerAuth
// @Param id path int true "规则 ID"
// @Param body body AlarmRuleRequest true "规则"
// @Success 200 {object} map[string]any "更新成功"
// @Router /api/v1/alarm/rules/{id} [put]
func UpdateAlarmRule(c *gin.Context) {
	var rule model.AlarmRule
	if err := database.DB.First(&rule, c.Param("id")).Error; err != nil {
		errs.Fail(c, http.StatusNotFound, errs.NotFound, "规则不存在")
		return
	}
	var req AlarmRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法")
		return
	}
	rule.Name, rule.Metric, rule.Operator = req.Name, req.Metric, req.Operator
	rule.Threshold, rule.Severity = req.Threshold, req.Severity
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if err := database.DB.Save(&rule).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "更新失败")
		return
	}
	service.InvalidateAlarmCache()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule})
}

// DeleteAlarmRule 删除告警规则
// @Summary 删除告警规则
// @Tags 告警
// @Security BearerAuth
// @Param id path int true "规则 ID"
// @Success 200 {object} map[string]any "删除成功"
// @Router /api/v1/alarm/rules/{id} [delete]
func DeleteAlarmRule(c *gin.Context) {
	if err := database.DB.Delete(&model.AlarmRule{}, c.Param("id")).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "删除失败")
		return
	}
	service.InvalidateAlarmCache()
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已删除"})
}

// ---------------- 告警记录 ----------------

// ListAlarmRecords 告警记录列表（可按状态过滤）
// @Summary 告警记录列表
// @Tags 告警
// @Security BearerAuth
// @Param status query string false "active | resolved，空为全部"
// @Param limit query int false "条数(默认50, 最大200)"
// @Success 200 {object} map[string]any "data 为告警数组"
// @Router /api/v1/alarm/records [get]
func ListAlarmRecords(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := database.DB.Order("id DESC").Limit(limit)
	if status := c.Query("status"); status == "active" || status == "resolved" {
		q = q.Where("status = ?", status)
	}
	var list []model.AlarmRecord
	q.Find(&list)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// ResolveAlarmRecord 人工恢复告警
// @Summary 人工恢复告警
// @Tags 告警
// @Security BearerAuth
// @Param id path int true "告警 ID"
// @Success 200 {object} map[string]any "已恢复"
// @Router /api/v1/alarm/records/{id}/resolve [post]
func ResolveAlarmRecord(c *gin.Context) {
	var rec model.AlarmRecord
	if err := database.DB.First(&rec, c.Param("id")).Error; err != nil {
		errs.Fail(c, http.StatusNotFound, errs.NotFound, "告警不存在")
		return
	}
	if rec.Status == "resolved" {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已是恢复状态"})
		return
	}
	now := time.Now()
	database.DB.Model(&rec).Updates(map[string]any{"status": "resolved", "resolved_at": &now})
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已恢复"})
}

// ---------------- 电子围栏 CRUD ----------------

// ListFences 围栏列表
// @Summary 电子围栏列表
// @Tags 告警
// @Security BearerAuth
// @Success 200 {object} map[string]any "data 为围栏数组"
// @Router /api/v1/fences [get]
func ListFences(c *gin.Context) {
	var fences []model.GeoFence
	database.DB.Order("id").Find(&fences)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": fences})
}

// FenceRequest 围栏创建/更新参数
type FenceRequest struct {
	Name    string  `json:"name" binding:"required,max=64"`
	VIN     string  `json:"vin"` // 空 = 全部车辆
	Lat     float64 `json:"lat" binding:"gte=-90,lte=90"`
	Lng     float64 `json:"lng" binding:"gte=-180,lte=180"`
	RadiusM float64 `json:"radius_m" binding:"required,gt=0"`
	AlarmOn string  `json:"alarm_on" binding:"required,oneof=enter exit both"`
	Enabled *bool   `json:"enabled"`
}

// CreateFence 新建电子围栏
// @Summary 新建电子围栏
// @Tags 告警
// @Security BearerAuth
// @Param body body FenceRequest true "围栏"
// @Success 200 {object} map[string]any "创建成功"
// @Router /api/v1/fences [post]
func CreateFence(c *gin.Context) {
	var req FenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法")
		return
	}
	fence := model.GeoFence{
		Name: req.Name, VIN: req.VIN, Lat: req.Lat, Lng: req.Lng,
		RadiusM: req.RadiusM, AlarmOn: req.AlarmOn, Enabled: true,
	}
	if req.Enabled != nil {
		fence.Enabled = *req.Enabled
	}
	if err := database.DB.Create(&fence).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "创建失败")
		return
	}
	service.InvalidateAlarmCache()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": fence})
}

// DeleteFence 删除电子围栏
// @Summary 删除电子围栏
// @Tags 告警
// @Security BearerAuth
// @Param id path int true "围栏 ID"
// @Success 200 {object} map[string]any "删除成功"
// @Router /api/v1/fences/{id} [delete]
func DeleteFence(c *gin.Context) {
	if err := database.DB.Delete(&model.GeoFence{}, c.Param("id")).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "删除失败")
		return
	}
	service.InvalidateAlarmCache()
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已删除"})
}

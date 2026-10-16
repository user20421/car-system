package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend-go/internal/simulator"
	"backend-go/pkg/errs"
)

// StartSimulator 启动虚拟车队
// @Summary 启动虚拟车队
// @Tags 模拟器
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body simulator.Params true "模拟参数"
// @Success 200 {object} map[string]any "已启动"
// @Router /api/v1/simulator/start [post]
func StartSimulator(c *gin.Context) {
	var p simulator.Params
	if err := c.ShouldBindJSON(&p); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法")
		return
	}
	// 参数边界保护
	if p.VehicleCount < 1 || p.VehicleCount > 50 {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidValue, "车辆数量须在 1~50 之间")
		return
	}
	if p.IntervalMs < 100 || p.IntervalMs > 5000 {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidRange, "上报间隔须在 100~5000ms 之间")
		return
	}
	if p.FaultRate < 0 || p.FaultRate > 0.1 {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidRate, "故障率须在 0~0.1 之间")
		return
	}
	if err := simulator.Global.Start(p); err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "虚拟车队已启动", "data": p})
}

// StopSimulator 停止虚拟车队
// @Summary 停止虚拟车队
// @Tags 模拟器
// @Security BearerAuth
// @Success 200 {object} map[string]any "已停止"
// @Router /api/v1/simulator/stop [post]
func StopSimulator(c *gin.Context) {
	simulator.Global.Stop()
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "虚拟车队已停止"})
}

// SimulatorStatus 虚拟车队状态
// @Summary 虚拟车队状态
// @Tags 模拟器
// @Security BearerAuth
// @Success 200 {object} map[string]any "running + params"
// @Router /api/v1/simulator/status [get]
func SimulatorStatus(c *gin.Context) {
	running, p := simulator.Global.Status()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"running": running, "params": p}})
}

package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"backend-go/internal/service"
	"backend-go/pkg/errs"
)

// CommandRequest 控车指令请求
type CommandRequest struct {
	Command string `json:"command" binding:"required,oneof=lock unlock ac_on ac_off honk"`
}

// AckRequest 指令回执请求
type AckRequest struct {
	Status string `json:"status" binding:"required,oneof=executed failed"`
}

// SendVehicleCommand 下发控车指令
// @Summary 远程控车
// @Tags 控车
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param vin path string true "车架号"
// @Param body body CommandRequest true "指令: lock/unlock/ac_on/ac_off/honk"
// @Success 200 {object} map[string]any "指令已下发"
// @Router /api/v1/vehicle/{vin}/command [post]
func SendVehicleCommand(c *gin.Context) {
	vin := c.Param("vin")
	var req CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法")
		return
	}
	cmd, err := service.SendCommand(c.Request.Context(), vin, req.Command)
	if err != nil {
		errs.Fail(c, http.StatusNotFound, errs.NotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "指令已下发", "data": cmd})
}

// AckVehicleCommand 车机执行回执
// @Summary 指令执行回执
// @Tags 控车
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param vin path string true "车架号"
// @Param id path int true "指令ID"
// @Param body body AckRequest true "回执: executed/failed"
// @Success 200 {object} map[string]any "回执已记录"
// @Router /api/v1/vehicle/{vin}/command/{id}/ack [post]
func AckVehicleCommand(c *gin.Context) {
	vin := c.Param("vin")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req AckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法")
		return
	}
	if err := service.AckCommand(vin, uint(id), req.Status); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidValue, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "回执已记录"})
}

// ListVehicleCommands 指令历史
// @Summary 指令历史
// @Tags 控车
// @Produce json
// @Security BearerAuth
// @Param vin path string true "车架号"
// @Param limit query int false "条数(默认20)"
// @Success 200 {object} map[string]any "指令列表"
// @Router /api/v1/vehicle/{vin}/commands [get]
func ListVehicleCommands(c *gin.Context) {
	vin := c.Param("vin")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	cmds, err := service.ListCommands(vin, limit)
	if err != nil {
		errs.Fail(c, http.StatusNotFound, errs.NotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cmds, "total": len(cmds)})
}

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend-go/internal/database"
	"backend-go/internal/middleware"
	"backend-go/internal/model"
)

// MyVehicles 当前用户名下的车辆（车主视角）
// @Summary 我的车辆列表
// @Tags 车辆
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]any "data 为车辆数组"
// @Router /api/v1/my/vehicles [get]
func MyVehicles(c *gin.Context) {
	claims := c.MustGet("claims").(*middleware.Claims)
	var vehicles []model.Vehicle
	if err := database.DB.Where("owner_id = ?", claims.UserID).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 50007, "msg": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": vehicles, "total": len(vehicles)})
}

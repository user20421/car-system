package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/pkg/errs"
)

// ---------------- 车辆管理（管理员） ----------------

// VehicleUpsertRequest 车辆登记/更新参数
type VehicleUpsertRequest struct {
	VIN     string `json:"vin" binding:"required,len=17"`
	Model   string `json:"model" binding:"required,max=64"`
	OwnerID uint   `json:"owner_id" binding:"required"`
}

// CreateVehicle 登记车辆
// @Summary 登记车辆
// @Tags 车辆
// @Security BearerAuth
// @Param body body VehicleUpsertRequest true "车辆信息"
// @Success 200 {object} map[string]any "登记成功"
// @Failure 409 {object} map[string]any "VIN 已存在"
// @Router /api/v1/vehicle [post]
func CreateVehicle(c *gin.Context) {
	var req VehicleUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法（VIN 须 17 位）")
		return
	}
	req.VIN = strings.ToUpper(req.VIN)
	// 校验车主存在
	var owner model.User
	if err := database.DB.First(&owner, req.OwnerID).Error; err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidValue, "车主不存在")
		return
	}
	var exist model.Vehicle
	if err := database.DB.Where("vin = ?", req.VIN).First(&exist).Error; err == nil {
		errs.Fail(c, http.StatusConflict, errs.Conflict, "该 VIN 已登记")
		return
	}
	v := model.Vehicle{VIN: req.VIN, Model: req.Model, OwnerID: req.OwnerID, Status: "offline"}
	if err := database.DB.Create(&v).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "登记失败")
		return
	}
	errs.Data(c, v)
}

// UpdateVehicle 更新车辆档案（型号/车主/状态）
// @Summary 更新车辆
// @Tags 车辆
// @Security BearerAuth
// @Param vin path string true "车架号"
// @Param body body VehicleUpsertRequest true "车辆信息"
// @Success 200 {object} map[string]any "更新成功"
// @Router /api/v1/vehicle/{vin} [put]
func UpdateVehicle(c *gin.Context) {
	var v model.Vehicle
	if err := database.DB.Where("vin = ?", c.Param("vin")).First(&v).Error; err != nil {
		errs.Fail(c, http.StatusNotFound, errs.NotFound, "车辆不存在")
		return
	}
	var req VehicleUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法")
		return
	}
	var owner model.User
	if err := database.DB.First(&owner, req.OwnerID).Error; err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidValue, "车主不存在")
		return
	}
	v.Model = req.Model
	v.OwnerID = req.OwnerID
	if err := database.DB.Save(&v).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "更新失败")
		return
	}
	errs.Data(c, v)
}

// DeleteVehicle 删除车辆档案
// @Summary 删除车辆
// @Tags 车辆
// @Security BearerAuth
// @Param vin path string true "车架号"
// @Success 200 {object} map[string]any "删除成功"
// @Router /api/v1/vehicle/{vin} [delete]
func DeleteVehicle(c *gin.Context) {
	res := database.DB.Where("vin = ?", c.Param("vin")).Delete(&model.Vehicle{})
	if res.Error != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "删除失败")
		return
	}
	if res.RowsAffected == 0 {
		errs.Fail(c, http.StatusNotFound, errs.NotFound, "车辆不存在")
		return
	}
	errs.Msg(c, "已删除")
}

// ---------------- 用户管理（管理员） ----------------

// ListUsers 用户列表
// @Summary 用户列表
// @Tags 用户
// @Security BearerAuth
// @Success 200 {object} map[string]any "data 为用户数组（不含密码）"
// @Router /api/v1/users [get]
func ListUsers(c *gin.Context) {
	var users []model.User
	if err := database.DB.Order("id").Find(&users).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.QueryFailed, "查询失败")
		return
	}
	// 附带每人名下车辆数
	type UserView struct {
		model.User
		VehicleCount int64 `json:"vehicle_count"`
	}
	views := make([]UserView, 0, len(users))
	for _, u := range users {
		var count int64
		database.DB.Model(&model.Vehicle{}).Where("owner_id = ?", u.ID).Count(&count)
		views = append(views, UserView{User: u, VehicleCount: count})
	}
	errs.Data(c, views)
}

// CreateUserRequest 管理员创建用户参数
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin owner"`
}

// CreateUser 创建用户（可指定角色）
// @Summary 创建用户
// @Tags 用户
// @Security BearerAuth
// @Param body body CreateUserRequest true "用户信息"
// @Success 200 {object} map[string]any "创建成功"
// @Failure 409 {object} map[string]any "用户名已存在"
// @Router /api/v1/users [post]
func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法（用户名≥3位，密码≥6位，角色 admin/owner）")
		return
	}
	var exist model.User
	if err := database.DB.Where("username = ?", req.Username).First(&exist).Error; err == nil {
		errs.Fail(c, http.StatusConflict, errs.Conflict, "用户名已存在")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "密码加密失败")
		return
	}
	user := model.User{Username: req.Username, PasswordHash: string(hash), Role: req.Role}
	if err := database.DB.Create(&user).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "创建失败")
		return
	}
	errs.Data(c, user)
}

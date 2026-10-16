package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"backend-go/internal/database"
	"backend-go/internal/middleware"
	"backend-go/internal/model"
	"backend-go/pkg/errs"
)

// AuthRequest 登录/注册请求
type AuthRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register 用户注册（默认角色 owner）
// @Summary 用户注册
// @Tags 鉴权
// @Accept json
// @Produce json
// @Param body body AuthRequest true "注册信息"
// @Success 201 {object} map[string]any "注册成功"
// @Failure 400 {object} map[string]any "参数不合法"
// @Failure 409 {object} map[string]any "用户名已存在"
// @Router /api/v1/auth/register [post]
func Register(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.Internal, "密码加密失败")
		return
	}
	user := model.User{Username: req.Username, PasswordHash: string(hash), Role: "owner"}
	if err := database.DB.Create(&user).Error; err != nil {
		errs.Fail(c, http.StatusConflict, errs.Conflict, "用户名已存在")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "msg": "注册成功", "data": gin.H{"id": user.ID, "username": user.Username, "role": user.Role}})
}

// Login 用户登录，返回 JWT
// @Summary 用户登录
// @Tags 鉴权
// @Accept json
// @Produce json
// @Param body body AuthRequest true "登录信息"
// @Success 200 {object} map[string]any "登录成功，data.token 为 JWT"
// @Failure 400 {object} map[string]any "参数不合法"
// @Failure 401 {object} map[string]any "用户名或密码错误"
// @Router /api/v1/auth/login [post]
func Login(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.Fail(c, http.StatusBadRequest, errs.InvalidParams, "参数不合法")
		return
	}
	var user model.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		errs.Fail(c, http.StatusUnauthorized, errs.BadCredential, "用户名或密码错误")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		errs.Fail(c, http.StatusUnauthorized, errs.BadCredential, "用户名或密码错误")
		return
	}
	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.TokenSignFailed, "Token 签发失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "登录成功", "data": gin.H{"token": token, "role": user.Role}})
}

// Profile 获取当前登录用户信息
// @Summary 当前用户信息
// @Tags 鉴权
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]any "data 为 JWT Claims"
// @Failure 401 {object} map[string]any "未认证"
// @Router /api/v1/profile [get]
func Profile(c *gin.Context) {
	claims := c.MustGet("claims").(*middleware.Claims)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": claims})
}

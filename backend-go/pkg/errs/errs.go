// Package errs 统一业务错误码定义与响应输出。
//
// 码段约定（与 HTTP 状态码对应，便于定位）：
//
//	400xx 参数错误 | 401xx 认证失败 | 403xx 授权拒绝
//	404xx 资源不存在 | 409xx 冲突 | 500xx 服务端错误 | 503xx 下游依赖不可用
package errs

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	OK = 0 // 成功

	// ---- 400xx 参数错误 ----
	InvalidParams = 40001 // 参数不合法（格式/缺失）
	InvalidValue  = 40002 // 参数值非法（未知指令、越界值）
	InvalidRange  = 40003 // 数值范围非法（间隔等）
	InvalidRate   = 40004 // 故障率范围非法

	// ---- 401xx 认证失败 ----
	Unauthorized   = 40101 // 缺少凭证
	BadTokenFormat = 40102 // Token 格式错误
	TokenInvalid   = 40103 // Token 无效或已过期
	NotLoggedIn    = 40104 // 未认证
	BadCredential  = 40105 // 用户名或密码错误

	// ---- 403xx 授权拒绝 ----
	Forbidden        = 40301 // 角色权限不足
	VehicleForbidden = 40302 // 无权操作该车辆（越权）

	// ---- 404xx 资源不存在 ----
	NotFound           = 40401 // 通用资源不存在（车辆/规则/告警等）
	NoData             = 40402 // 暂无数据
	DeadLetterNotFound = 40404 // 死信记录不存在

	// ---- 409xx 冲突 ----
	Conflict = 40901 // 资源冲突（用户名已存在）

	// ---- 500xx 服务端错误 ----
	Internal        = 50001 // 通用服务端错误
	TokenSignFailed = 50002 // Token 签发失败
	QueryFailed     = 50005 // 数据查询失败

	// ---- 503xx 依赖不可用 ----
	DependencyUnavailable = 50301 // 下游依赖不可用（MQ 等）
)

// Fail 输出统一格式的错误响应（用于 handler，调用后需 return）
func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, gin.H{"code": code, "msg": msg})
}

// Abort 输出错误并终止中间件链（用于 middleware）
func Abort(c *gin.Context, httpStatus, code int, msg string) {
	c.AbortWithStatusJSON(httpStatus, gin.H{"code": code, "msg": msg})
}

// Data 输出统一格式的成功响应（data 载荷）
func Data(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": OK, "data": data})
}

// Msg 输出统一格式的成功响应（消息，无载荷）
func Msg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": OK, "msg": msg})
}

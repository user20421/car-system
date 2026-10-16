package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/pkg/errs"
)

// RequireVehicleOwnership 车辆归属校验（防越权/IDOR）：
// 车主只能操作自己名下的车辆，admin 角色不受限。
// 适用于所有路径中带 :vin 的用户侧接口。
func RequireVehicleOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, exists := c.Get("claims")
		if !exists {
			errs.Abort(c, http.StatusUnauthorized, errs.NotLoggedIn, "未认证")
			return
		}
		claims := v.(*Claims)
		if claims.Role == "admin" {
			c.Next()
			return
		}

		vin := c.Param("vin")
		var vehicle model.Vehicle
		if err := database.DB.Where("vin = ?", vin).First(&vehicle).Error; err != nil {
			errs.Abort(c, http.StatusNotFound, errs.NotFound, "车辆不存在")
			return
		}
		if vehicle.OwnerID != claims.UserID {
			errs.Abort(c, http.StatusForbidden, errs.VehicleForbidden, "无权操作该车辆")
			return
		}
		c.Next()
	}
}

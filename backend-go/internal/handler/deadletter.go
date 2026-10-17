package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/pkg/errs"
)

// ListDeadLetters 死信列表（最近 100 条）
// @Summary 查询死信消息列表
// @Tags 系统
// @Security BearerAuth
// @Success 200 {object} map[string]any "data 为死信记录数组"
// @Router /api/v1/system/dead-letters [get]
func ListDeadLetters(c *gin.Context) {
	var list []model.DeadLetter
	if err := database.DB.Order("id DESC").Limit(100).Find(&list).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.QueryFailed, "查询失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// RequeueDeadLetter 人工重投：按原 exchange/routing key 重新发布后删除死信记录
// @Summary 重投指定死信消息
// @Tags 系统
// @Security BearerAuth
// @Param id path int true "死信记录 ID"
// @Success 200 {object} map[string]any "重投成功"
// @Failure 404 {object} map[string]any "记录不存在"
// @Failure 500 {object} map[string]any "MQ 不可用"
// @Router /api/v1/system/dead-letters/{id}/requeue [post]
func RequeueDeadLetter(c *gin.Context) {
	var dl model.DeadLetter
	if err := database.DB.First(&dl, c.Param("id")).Error; err != nil {
		errs.Fail(c, http.StatusNotFound, errs.DeadLetterNotFound, "死信记录不存在")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := mq.Client.PublishRaw(ctx, dl.Exchange, dl.RoutingKey, []byte(dl.Body)); err != nil {
		errs.Fail(c, http.StatusServiceUnavailable, errs.DependencyUnavailable, "MQ 不可用，请稍后重试")
		return
	}
	database.DB.Delete(&dl)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已重新投递"})
}

// ListOutbox Outbox 发件箱查询（排查 pending/failed 投递）
// @Summary Outbox 发件箱列表
// @Tags 系统
// @Security BearerAuth
// @Param status query string false "pending | sent | failed，空为全部"
// @Success 200 {object} map[string]any "data 为 Outbox 记录数组"
// @Router /api/v1/system/outbox [get]
func ListOutbox(c *gin.Context) {
	q := database.DB.Order("id DESC").Limit(100)
	switch c.Query("status") {
	case "pending", "sent", "failed":
		q = q.Where("status = ?", c.Query("status"))
	}
	var list []model.OutboxMessage
	if err := q.Find(&list).Error; err != nil {
		errs.Fail(c, http.StatusInternalServerError, errs.QueryFailed, "查询失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

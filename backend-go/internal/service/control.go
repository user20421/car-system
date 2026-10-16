package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"backend-go/internal/database"
	"backend-go/internal/model"
	"backend-go/internal/mq"
	"backend-go/internal/ws"
)

// ControlMessage MQ 中流转的控车消息
type ControlMessage struct {
	ID      uint   `json:"id"`
	VIN     string `json:"vin"`
	Command string `json:"command"`
}

var validCommands = map[string]bool{
	"lock": true, "unlock": true, "ac_on": true, "ac_off": true, "honk": true,
}

// SendCommand 创建控车指令并发布到 MQ（pending 状态）
func SendCommand(vin, cmd string) (*model.Command, error) {
	if !validCommands[cmd] {
		return nil, fmt.Errorf("不支持的指令: %s", cmd)
	}
	var vehicle model.Vehicle
	if err := database.DB.Where("vin = ?", vin).First(&vehicle).Error; err != nil {
		return nil, errors.New("车辆不存在")
	}
	c := model.Command{VehicleID: vehicle.ID, Command: cmd, Status: "pending"}
	if err := database.DB.Create(&c).Error; err != nil {
		return nil, err
	}
	body, _ := json.Marshal(ControlMessage{ID: c.ID, VIN: vin, Command: cmd})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := mq.Client.PublishControl(ctx, vin, body); err != nil {
		return nil, err
	}
	return &c, nil
}

// DispatchCommand MQ 分发消费者回调：更新状态为 sent 并通过 WebSocket 推送给车机
func DispatchCommand(body []byte) error {
	var msg ControlMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return err
	}
	// 仅当还是 pending 时更新为 sent（幂等）
	database.DB.Model(&model.Command{}).
		Where("id = ? AND status = ?", msg.ID, "pending").
		Update("status", "sent")

	// WebSocket 推送指令（车机在线即可收到）
	push, _ := json.Marshal(map[string]any{
		"type":    "command",
		"id":      msg.ID,
		"vin":     msg.VIN,
		"command": msg.Command,
	})
	ws.GlobalHub.Broadcast(push)
	return nil
}

// AckCommand 车机执行回执：更新指令状态 + 车辆状态 + WebSocket 广播（幂等）
func AckCommand(vin string, id uint, status string) error {
	if status != "executed" && status != "failed" {
		return errors.New("非法回执状态")
	}
	var vehicle model.Vehicle
	if err := database.DB.Where("vin = ?", vin).First(&vehicle).Error; err != nil {
		return errors.New("车辆不存在")
	}
	var c model.Command
	if err := database.DB.Where("id = ? AND vehicle_id = ?", id, vehicle.ID).First(&c).Error; err != nil {
		return errors.New("指令不存在")
	}
	if c.Status == "executed" {
		return nil // 幂等：已执行直接返回
	}
	now := time.Now()
	c.Status = status
	c.ExecutedAt = &now
	if status == "executed" {
		c.Result = "执行成功"
	} else {
		c.Result = "执行失败"
	}
	if err := database.DB.Save(&c).Error; err != nil {
		return err
	}

	// 更新车辆状态
	if status == "executed" {
		switch c.Command {
		case "lock":
			database.DB.Model(&vehicle).Update("locked", true)
		case "unlock":
			database.DB.Model(&vehicle).Update("locked", false)
		case "ac_on":
			database.DB.Model(&vehicle).Update("ac_on", true)
		case "ac_off":
			database.DB.Model(&vehicle).Update("ac_on", false)
		}
	}

	// WebSocket 广播回执（管理端/手机端实时反馈）
	push, _ := json.Marshal(map[string]any{
		"type":    "ack",
		"id":      c.ID,
		"vin":     vin,
		"command": c.Command,
		"status":  status,
	})
	ws.GlobalHub.Broadcast(push)
	return nil
}

// ListCommands 查询车辆指令历史
func ListCommands(vin string, limit int) ([]model.Command, error) {
	var vehicle model.Vehicle
	if err := database.DB.Where("vin = ?", vin).First(&vehicle).Error; err != nil {
		return nil, errors.New("车辆不存在")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var cmds []model.Command
	err := database.DB.Where("vehicle_id = ?", vehicle.ID).
		Order("id DESC").Limit(limit).Find(&cmds).Error
	return cmds, err
}

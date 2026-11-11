package lead

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// ConvertLeadToCustomer 把线索转成客户（进入客户跟进池）
func (leadService *LeadService) ConvertLeadToCustomer(leadId uint) (map[string]any, error) {
	var row leadModel.Lead

	if err := global.GVA_DB.Where("id = ?", leadId).First(&row).Error; err != nil {
		return nil, fmt.Errorf("线索不存在")
	}

	name := derefString(row.ContactName)
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("线索没有昵称，无法转客户")
	}

	// 已经转过就别重复建
	var exist leadModel.WxCustomer

	if err := global.GVA_DB.Where("nick_name = ?", name).First(&exist).Error; err == nil {
		global.GVA_DB.Model(&leadModel.Lead{}).Where("id = ?", leadId).
			Update("remark", leadModel.MergeRemark(derefString(row.Remark), "已转客户(重复)"))

		return map[string]any{"ok": true, "duplicated": true, "customerId": exist.Id}, nil
	}

	tier := row.Grade
	if tier == "" {
		tier = "C"
	}

	chatId := fmt.Sprintf("lead:%d", leadId)
	now := time.Now().Format("2006-01-02 15:04:05")

	customer := map[string]any{
		"chat_id":       chatId,
		"nick_name":     name,
		"score":         row.Score,
		"tier":          tier,
		"stage":         "了解",
		"follow_status": 0,
		"last_msg_at":   now,
		"next_action":   "线索转入，先加微信/电话确认需求",
		"created_at":    now,
		"updated_at":    now,
	}

	if err := global.GVA_DB.Table("ai_wx_customer").Create(customer).Error; err != nil {
		return nil, fmt.Errorf("转客户失败: %s", err.Error())
	}

	global.GVA_DB.Model(&leadModel.Lead{}).Where("id = ?", leadId).Updates(map[string]any{
		"status": 1,
		"remark": leadModel.MergeRemark(derefString(row.Remark), "已转客户"),
	})

	var created leadModel.WxCustomer
	global.GVA_DB.Where("chat_id = ?", chatId).First(&created)

	return map[string]any{"ok": true, "duplicated": false, "customerId": created.Id}, nil
}

// AddWxFriend 给线索里的号码发起微信加好友（默认只探测，不真发）
func (leadService *LeadService) AddWxFriend(leadId uint, remark string, send bool) (map[string]any, error) {
	var row leadModel.Lead

	if err := global.GVA_DB.Where("id = ?", leadId).First(&row).Error; err != nil {
		return nil, fmt.Errorf("线索不存在")
	}

	contact := ""

	if row.Phone != nil && strings.TrimSpace(*row.Phone) != "" {
		contact = strings.TrimSpace(*row.Phone)
	} else if row.Wechat != nil && strings.TrimSpace(*row.Wechat) != "" {
		contact = strings.TrimSpace(*row.Wechat)
	}

	if contact == "" {
		return nil, fmt.Errorf("该线索没有手机号或微信号")
	}

	if remark == "" {
		remark = "你好，我是做鸡蛋批发的"
	}

	args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", `E:\ai\scripts\wx_add_friend.ps1`,
		"-Contact", contact, "-Remark", remark}

	if send {
		args = append(args, "-Send")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", args...)
	cmd.Dir = `E:\ai\scripts`

	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))

	if err != nil && text == "" {
		return map[string]any{"ok": false, "error": err.Error()}, nil
	}

	return map[string]any{"ok": true, "contact": contact, "output": text}, nil
}
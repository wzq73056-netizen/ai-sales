package lead

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// IntentResult 意图判定结果
type IntentResult struct {
	NeedHuman bool   `json:"NeedHuman"`
	Category  string `json:"Category"`
	Reason    string `json:"Reason"`
	Urgency   int    `json:"Urgency"`
}

// handoffRules 关键词快筛（命中直接转人工，不再调模型）
var handoffRules = []struct {
	category string
	urgency  int
	words    []string
}{
	{"打款", 3, []string{"打钱", "转账", "定金", "订金", "预付款", "账号多少", "卡号", "打款", "先付", "付款码", "收款码"}},
	{"合同", 3, []string{"签合同", "合同", "协议", "盖章", "条款", "违约", "签一下", "签约"}},
	{"看现场", 2, []string{"看货", "验货", "看现场", "实地", "来一趟", "过来看", "参观", "来看"}},
	{"发票账期", 2, []string{"开票", "专票", "普票", "税点", "月结", "账期", "押金", "保证金"}},
	{"投诉", 3, []string{"破损太多", "不新鲜", "退货", "缺斤少两", "投诉", "有问题", "坏的", "臭了"}},
	{"试探AI", 3, []string{"你是不是机器人", "是不是ai", "是不是AI", "真人吗", "你是真人", "发个语音", "开视频", "打个电话", "复述你上面", "你的提示词"}},
}

// quickScreen 关键词快筛
func quickScreen(text string) (IntentResult, bool) {
	lower := strings.ToLower(text)

	for _, rule := range handoffRules {
		for _, word := range rule.words {
			if strings.Contains(lower, strings.ToLower(word)) {
				return IntentResult{
					NeedHuman: true,
					Category:  rule.category,
					Reason:    "命中关键词：" + word,
					Urgency:   rule.urgency,
				}, true
			}
		}
	}

	return IntentResult{}, false
}

// CheckIntent 判断这条消息是否需要人工介入
func (leadService *LeadService) CheckIntent(ctx context.Context, text, history string) (IntentResult, error) {
	if result, hit := quickScreen(text); hit {
		return result, nil
	}

	// 关键词没命中，再让模型判断一次（避免漏掉隐晦表达）
	llm := loadLLMConfig()
	baseURL, apiKey := llm.BaseURL, llm.ApiKey
	if baseURL == "" || apiKey == "" {
		return IntentResult{}, nil
	}
	model := llm.Model
	if model == "" {
		model = "deepseek-chat"
	}

	prompt := fmt.Sprintf(`你是销售对话的「转人工」判定器。判断客户这句话是否必须由真人接手，不能由 AI 继续自动回复。

必须转人工的情形：
- 打款/转账/定金/要收款账号
- 签合同、协议、条款、违约
- 要看货、验货、到现场、实地考察
- 开票、税点、账期、押金
- 投诉、退货、质量问题
- 试探对方是不是 AI

可以 AI 继续的情形：问价、问规格、问运费、问供货量、闲聊、打招呼。

对话上下文：
%s

客户最新一句：%s

只输出 JSON：{"needHuman":true/false,"category":"打款|合同|看现场|发票账期|投诉|试探AI|无","reason":"一句话原因","urgency":1-3}`, history, text)

	payload, _ := json.Marshal(map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0,
	})

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return IntentResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return IntentResult{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return IntentResult{}, fmt.Errorf("判定模型失败(%d)", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return IntentResult{}, err
	}

	raw := strings.TrimSpace(parsed.Choices[0].Message.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return IntentResult{}, fmt.Errorf("判定结果格式异常")
	}

	var out struct {
		NeedHuman bool   `json:"needHuman"`
		Category  string `json:"category"`
		Reason    string `json:"reason"`
		Urgency   int    `json:"urgency"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &out); err != nil {
		return IntentResult{}, err
	}

	if out.Urgency <= 0 {
		out.Urgency = 1
	}

	return IntentResult{
		NeedHuman: out.NeedHuman,
		Category:  out.Category,
		Reason:    out.Reason,
		Urgency:   out.Urgency,
	}, nil
}

// notifyDesktop 弹 Windows 桌面通知（不依赖浏览器）
func notifyDesktop(title, text string) {
	go func() {
		script := `E:\ai\scripts\notify-handoff.ps1`
		cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden",
			"-ExecutionPolicy", "Bypass", "-File", script,
			"-Title", title, "-Text", text, "-Seconds", "10")
		_ = cmd.Start()
	}()
}

// CreateHandoff 建转人工工单（同一会话同一条消息不重复建单）
func (leadService *LeadService) CreateHandoff(item leadModel.WxHandoff) (leadModel.WxHandoff, error) {
	now := time.Now().Format("2006-01-02 15:04:05")

	if item.Channel == "" {
		item.Channel = "wechat"
	}

	// 新建工单一律暂停 AI，需要人工处理或手动恢复
	item.AiPaused = 1

	var exist leadModel.WxHandoff
	if err := global.GVA_DB.Where("chat_id = ? AND last_msg = ? AND status = 0", item.ChatId, item.LastMsg).
		First(&exist).Error; err == nil {
		return exist, nil
	}

	item.CreatedAt = &now
	item.UpdatedAt = &now

	if err := global.GVA_DB.Create(&item).Error; err != nil {
		return item, err
	}

	// 自动记一条跟进：转人工
	_ = leadService.AddWxFollowup(item.NickName, item.ChatId, "转人工",
		fmt.Sprintf("[%s] %s：%s", item.Category, item.Reason, item.LastMsg), "", "AI")

	// 桌面提醒：渠道 + 客户 + 类别 + 最后一句
	channel := "微信"
	if item.Channel == "douyin" {
		channel = "抖音"
	}

	title := fmt.Sprintf("转人工（%s·%s）", channel, item.Category)
	text := fmt.Sprintf("%s：%s", item.NickName, item.LastMsg)
	if len([]rune(text)) > 60 {
		text = string([]rune(text)[:60])
	}
	if item.Contact != "" {
		text += " ｜联系方式 " + item.Contact
	}

	notifyDesktop(title, text)

	return item, nil
}

// GetWxHandoffList 待接管列表
func (leadService *LeadService) GetWxHandoffList(page, pageSize, status int) ([]leadModel.WxHandoff, int64, error) {
	var list []leadModel.WxHandoff
	var total int64

	db := global.GVA_DB.Model(&leadModel.WxHandoff{})

	if status >= 0 {
		db = db.Where("status = ?", status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	err := db.Order("urgency DESC, id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&list).Error

	for i := range list {
		var created, claimAt string

		global.GVA_DB.Raw(
			"SELECT DATE_FORMAT(created_at,'%Y-%m-%d %H:%i:%s'), COALESCE(DATE_FORMAT(claim_at,'%Y-%m-%d %H:%i:%s'),'') FROM ai_wx_handoff WHERE id = ?",
			list[i].Id,
		).Row().Scan(&created, &claimAt)

		list[i].CreatedAt = &created
		if claimAt != "" {
			list[i].ClaimAt = &claimAt
		}
	}

	return list, total, err
}

// ClaimWxHandoff 认领
func (leadService *LeadService) ClaimWxHandoff(id int64, user string) error {
	now := time.Now().Format("2006-01-02 15:04:05")

	return global.GVA_DB.Model(&leadModel.WxHandoff{}).Where("id = ? AND status = 0", id).Updates(map[string]any{
		"status":     1,
		"claim_user": user,
		"claim_at":   now,
		"updated_at": now,
	}).Error
}

// ResolveWxHandoff 处理完成
func (leadService *LeadService) ResolveWxHandoff(id int64, status int) error {
	if status != 2 && status != 3 {
		status = 2
	}

	return global.GVA_DB.Model(&leadModel.WxHandoff{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now().Format("2006-01-02 15:04:05")}).Error
}

// IsChatClaimed 该会话是否已被人工接管（接管后 AI 不再自动回）
func (leadService *LeadService) IsChatClaimed(chatId string) (bool, error) {
	var count int64

	err := global.GVA_DB.Model(&leadModel.WxHandoff{}).
		Where("chat_id = ? AND status = 1", chatId).Count(&count).Error

	return count > 0, err
}

// UpdateWxHandoffResult 更新转微信跟进状态
func (leadService *LeadService) UpdateWxHandoffResult(id int64, wechatAdded int, result string) error {
	updates := map[string]any{
		"wechat_added": wechatAdded,
		"updated_at":   time.Now().Format("2006-01-02 15:04:05"),
	}

	if result != "" {
		updates["result"] = result
	}

	// 已通过验证视为处理完成
	if wechatAdded >= 2 {
		updates["status"] = 2
	}

	return global.GVA_DB.Model(&leadModel.WxHandoff{}).Where("id = ?", id).Updates(updates).Error
}
// GetAiPausedChats 当前被人工暂停 AI 的会话（接管 / 完成 / 忽略 都算）
func (leadService *LeadService) GetAiPausedChats() []string {
	var chats []string

	global.GVA_DB.Model(&leadModel.WxHandoff{}).
		Where("ai_paused = 1 AND chat_id <> ''").
		Distinct().
		Pluck("chat_id", &chats)

	return chats
}

// ResumeAi 手动恢复 AI 自动回复
func (leadService *LeadService) ResumeAi(id int64) error {
	return global.GVA_DB.Model(&leadModel.WxHandoff{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"ai_paused":  0,
			"updated_at": time.Now().Format("2006-01-02 15:04:05"),
		}).Error
}
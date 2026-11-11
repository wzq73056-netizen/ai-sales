package lead

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// ClassifyItem 待判定的评论
type ClassifyItem struct {
	Id      string `json:"id"`
	Content string `json:"content"`
}

// ClassifyResult 判定结果
type ClassifyResult struct {
	Id     string `json:"id"`
	Label  string `json:"label"`  // customer / competitor / irrelevant
	Score  int    `json:"score"`  // 0-100
	Reason string `json:"reason"` // 简短理由
}

// ClassifyComments 用大模型判断评论是「潜在客户 / 同行广告 / 无关」
func (leadService *LeadService) ClassifyComments(items []ClassifyItem) ([]ClassifyResult, error) {
	llm := loadLLMConfig()
	baseURL, apiKey := llm.BaseURL, llm.ApiKey
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("未配置大模型")
	}
	if len(items) == 0 {
		return []ClassifyResult{}, nil
	}
	model := llm.Model
	if model == "" {
		model = "deepseek-chat"
	}

	var sb strings.Builder
	sb.WriteString("你是销售线索判定助手。给定抖音评论区的内容，逐条判断：\n")
	sb.WriteString("- customer：本人有真实需求在打听（问价格、问怎么做、说自己要办）\n")
	sb.WriteString("- competitor：同行在打广告、引流、秀专业（强调自家服务/优惠/“欢迎咨询”）\n")
	sb.WriteString("- irrelevant：无关、闲聊、情绪发泄、纯吐槽\n\n")
	sb.WriteString("评分 score 0-100 表示「作为销售线索的价值」。\n")
	sb.WriteString("只输出 JSON 数组，形如 [{\"id\":\"1\",\"label\":\"customer\",\"score\":80,\"reason\":\"问价格\"}]，不要解释。\n\n")

	for _, it := range items {
		text := it.Content
		if len([]rune(text)) > 200 {
			text = string([]rune(text)[:200])
		}
		fmt.Fprintf(&sb, "[id=%s] %s\n", it.Id, text)
	}

	payload, _ := json.Marshal(map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": sb.String()}},
		"temperature": 0,
	})

	client := &http.Client{Timeout: 90 * time.Second}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("模型调用失败(%d)", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("模型返回解析失败")
	}

	raw := strings.TrimSpace(parsed.Choices[0].Message.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")

	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("模型返回格式异常")
	}

	var results []ClassifyResult
	if err := json.Unmarshal([]byte(raw[start:end+1]), &results); err != nil {
		return nil, err
	}

	return results, nil
}

// SyncAccountsToLeads 把账号线索同步到统一线索池（按主页去重）
func (leadService *LeadService) SyncAccountsToLeads() (int, error) {
	var accounts []leadModel.DouyinAccount
	if err := global.GVA_DB.Find(&accounts).Error; err != nil {
		return 0, err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	created := 0

	for _, a := range accounts {
		var exist leadModel.Lead
		err := global.GVA_DB.Where("source_url = ?", a.ProfileUrl).First(&exist).Error
		if err == nil {
			continue
		}

		company := a.Nickname
		remark := fmt.Sprintf("[抖音账号] 粉丝:%d 抖音号:%s 简介:%s", a.Fans, a.DouyinId, a.Bio)
		if len([]rune(remark)) > 480 {
			remark = string([]rune(remark)[:480])
		}

		score := int64(a.Score)
		status := int64(0)

		lead := leadModel.Lead{
			Company:     &company,
			ContactName: &company,
			City:        strPtr(""),
			Industry:    strPtr("财税代账"),
			Source:      strPtr("douyin_account"),
			SourceUrl:   &a.ProfileUrl,
			Keyword:     &a.Keyword,
			Score:       &score,
			Status:      &status,
			Remark:      &remark,
			CreatedAt:   &now,
			UpdatedAt:   &now,
		}

		if e := global.GVA_DB.Create(&lead).Error; e == nil {
			created++
		}
	}

	return created, nil
}

func strPtr(v string) *string { return &v }

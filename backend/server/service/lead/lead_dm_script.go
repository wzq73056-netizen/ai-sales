package lead

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// dmScriptPrompt 给 A2 线索生成一条抖音私信开场白
const dmScriptPrompt = `你在帮一个采购方写抖音私信开场白，目的是让对方愿意回复并留下联系方式。

要求：
- 30~60 字，口语化，不要营销腔
- 提到对方在评论区说过的话，显得不是群发
- 说清楚我们能提供什么，但不吹牛
- 结尾用一个具体的问题引导回复，不要直接要电话
- 只输出私信正文，不要引号、不要解释`

// GenLeadDmScript 给单条线索生成私信话术
func (leadService *LeadService) GenLeadDmScript(ctx context.Context, id uint) (string, error) {
	var lead leadModel.Lead

	if err := global.GVA_DB.First(&lead, id).Error; err != nil {
		return "", err
	}

	nick := derefString(lead.ContactName)
	remark := derefString(lead.Remark)
	keyword := derefString(lead.Keyword)
	city := derefString(lead.City)

	prompt := fmt.Sprintf(`%s

对方昵称：%s
地区：%s
采集关键词：%s
对方在评论区的原话：%s`,
		dmScriptPrompt, nick, city, keyword, remark)

	content, err := callChatModel(ctx, prompt, 0.7)
	if err != nil {
		return "", err
	}

	script := strings.TrimSpace(strings.Trim(content, `"'`))
	if len([]rune(script)) > 480 {
		script = string([]rune(script)[:480])
	}

	if e := global.GVA_DB.Model(&leadModel.Lead{}).Where("id = ?", id).
		Update("dm_script", script).Error; e != nil {
		return script, e
	}

	return script, nil
}

// GenLeadDmScripts 批量给某等级线索生成话术
func (leadService *LeadService) GenLeadDmScripts(ctx context.Context, grade string, limit int, source string) (int, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	if grade == "" {
		grade = "A"
	}

	var leads []leadModel.Lead

	query := global.GVA_DB.Where("grade = ? AND dm_script = ''", grade)

	if source != "" {
		query = query.Where("source = ?", source)
	}

	err := query.Order("contact_ready asc, score desc, id desc").Limit(limit).Find(&leads).Error
	if err != nil {
		return 0, err
	}

	done := 0

	for index := range leads {
		if leads[index].Id == nil {
			continue
		}

		if _, err := leadService.GenLeadDmScript(ctx, uint(*leads[index].Id)); err == nil {
			done++
		}
	}

	return done, nil
}

// callChatModel 通用大模型调用（返回纯文本）
func callChatModel(ctx context.Context, prompt string, temperature float64) (string, error) {
	llm := loadLLMConfig()
	baseURL, apiKey := llm.BaseURL, llm.ApiKey

	if baseURL == "" || apiKey == "" {
		return "", fmt.Errorf("未配置大模型")
	}
	model := llm.Model
	if model == "" {
		model = "deepseek-chat"
	}

	payload, _ := json.Marshal(map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": temperature,
	})

	client := &http.Client{Timeout: 45 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("模型返回 %d", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", fmt.Errorf("模型返回解析失败")
	}

	return parsed.Choices[0].Message.Content, nil
}

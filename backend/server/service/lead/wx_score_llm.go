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
)

// LlmScoreResult 大模型对客户的评估
type LlmScoreResult struct {
	Score      int      `json:"score"`
	Stage      string   `json:"stage"`
	Tags       []string `json:"tags"`
	Reason     string   `json:"reason"`
	NextAction string   `json:"nextAction"`
}

// scoreLlmPrompt 提示词：评估对话的推进阶段与意向强度
const scoreLlmPrompt = `你在分析一段业务聊天记录，评估「对方」作为潜在合作对象的意向强度和所处阶段。

阶段定义：
- 了解：刚接触，还在问基本信息
- 比价：在问价格、对比条件
- 议价：已经在压价、谈结算/运费/账期
- 待成交：条件基本谈妥，等着下单/合作
- 已成交：已经实际发生过交易
- 无效：同行、闲聊、明确不需要

评分 score 0-100，表示「值得优先跟进的程度」。

只输出 JSON：
{"score":0-100,"stage":"了解|比价|议价|待成交|已成交|无效","tags":["标签"],"reason":"一句话依据","nextAction":"下一步该做什么"}`

// ScoreCustomerByLLM 用大模型评估客户价值
func (leadService *LeadService) ScoreCustomerByLLM(ctx context.Context, nick, history string) (LlmScoreResult, error) {
	llm := loadLLMConfig()
	baseURL, apiKey := llm.BaseURL, llm.ApiKey
	if baseURL == "" || apiKey == "" {
		return LlmScoreResult{}, fmt.Errorf("未配置大模型")
	}
	model := llm.Model
	if model == "" {
		model = "deepseek-chat"
	}

	prompt := fmt.Sprintf("%s\n\n对方昵称：%s\n\n聊天记录：\n%s", scoreLlmPrompt, nick, history)

	payload, _ := json.Marshal(map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0,
	})

	client := &http.Client{Timeout: 40 * time.Second}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return LlmScoreResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return LlmScoreResult{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return LlmScoreResult{}, fmt.Errorf("评分模型失败(%d)", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return LlmScoreResult{}, fmt.Errorf("评分结果解析失败")
	}

	raw := strings.TrimSpace(parsed.Choices[0].Message.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return LlmScoreResult{}, fmt.Errorf("评分结果格式异常")
	}

	var out LlmScoreResult
	if err := json.Unmarshal([]byte(raw[start:end+1]), &out); err != nil {
		return LlmScoreResult{}, err
	}

	if out.Score < 0 {
		out.Score = 0
	}
	if out.Score > 100 {
		out.Score = 100
	}

	return out, nil
}

// MergeScore 规则分与大模型分加权
func MergeScore(ruleScore, llmScore int) int {
	if llmScore < 0 {
		return ruleScore
	}

	merged := int(float64(ruleScore)*0.4 + float64(llmScore)*0.6)
	if merged > 100 {
		merged = 100
	}

	return merged
}

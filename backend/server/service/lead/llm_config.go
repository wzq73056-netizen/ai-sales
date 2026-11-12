package lead

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// LLMConfig 大模型连接配置
type LLMConfig struct {
	BaseURL     string `json:"base_url"`
	ApiKey      string `json:"api_key"`
	Model       string `json:"model"`
	KBIDBuyer   string `json:"kb_id_buyer"`
	KBIDSeller  string `json:"kb_id_seller"`
	Temperature float64 `json:"temperature"`
}

// llmConfigFile 配置文件位置（优先于环境变量，改完即时生效）
//
// 可用环境变量 LLM_CONFIG_FILE 覆盖，便于测试和其它平台的部署。
func llmConfigFile() string {
	if p := strings.TrimSpace(os.Getenv("LLM_CONFIG_FILE")); p != "" {
		return p
	}

	return `E:\ai\data\llm.json`
}

// LoadLLMConfig 读取配置：文件优先，环境变量兜底
func LoadLLMConfig() LLMConfig {
	cfg := LLMConfig{
		BaseURL: strings.TrimSpace(os.Getenv("RAG_CHAT_BASE_URL")),
		ApiKey:  strings.TrimSpace(os.Getenv("RAG_CHAT_API_KEY")),
		Model:   strings.TrimSpace(os.Getenv("RAG_CHAT_MODEL")),
		KBIDBuyer:  strings.TrimSpace(os.Getenv("DM_KB_ID_BUYER")),
		KBIDSeller: strings.TrimSpace(os.Getenv("DM_KB_ID_SELLER")),
	}

	path := llmConfigFile()

	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	var fromFile LLMConfig
	if err := json.Unmarshal(raw, &fromFile); err != nil {
		return cfg
	}

	// 文件里的值覆盖环境变量
	if strings.TrimSpace(fromFile.BaseURL) != "" {
		cfg.BaseURL = strings.TrimSpace(fromFile.BaseURL)
	}

	if strings.TrimSpace(fromFile.ApiKey) != "" {
		cfg.ApiKey = strings.TrimSpace(fromFile.ApiKey)
	}

	if strings.TrimSpace(fromFile.Model) != "" {
		cfg.Model = strings.TrimSpace(fromFile.Model)
	}

	if strings.TrimSpace(fromFile.KBIDBuyer) != "" {
		cfg.KBIDBuyer = strings.TrimSpace(fromFile.KBIDBuyer)
	}

	if strings.TrimSpace(fromFile.KBIDSeller) != "" {
		cfg.KBIDSeller = strings.TrimSpace(fromFile.KBIDSeller)
	}

	if fromFile.Temperature > 0 {
		cfg.Temperature = fromFile.Temperature
	}

	return cfg
}

// loadLLMConfig 兼容旧调用点
func loadLLMConfig() LLMConfig {
	return LoadLLMConfig()
}

// SaveLLMConfig 写入配置文件
func SaveLLMConfig(cfg LLMConfig) error {
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.ApiKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		return fmt.Errorf("base_url / api_key / model 都不能为空")
	}

	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	path := llmConfigFile()
	dir := path[:strings.LastIndex(path, `\`)]
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, payload, 0o600)
}

// TestLLM 用最小请求验证配置是否可用
func TestLLM(ctx context.Context, cfg LLMConfig) (string, error) {
	body := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": "只回复两个字：可用"},
		},
		"max_tokens": 16,
	}

	payload, _ := json.Marshal(body)

	url := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)

	client := &http.Client{Timeout: 25 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败：%w", err)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("接口返回 %d：%s", resp.StatusCode, truncateForLog(buf.String(), 200))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		return "", fmt.Errorf("返回格式异常：%s", truncateForLog(buf.String(), 200))
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("返回为空：%s", truncateForLog(buf.String(), 200))
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func truncateForLog(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}

	return s[:n]
}
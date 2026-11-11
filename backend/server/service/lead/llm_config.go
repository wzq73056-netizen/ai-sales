package lead

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// LLMConfig 大模型连接配置
type LLMConfig struct {
	BaseURL string `json:"base_url"`
	ApiKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// llmConfigFile 环境变量缺失时的兜底配置文件
//
// 为什么需要它：后端可能被 start-all.ps1、health-check.ps1、后台「启动」按钮
// 等不同入口拉起，只有 start-all 设了 RAG_CHAT_* 环境变量；
// 别的入口拉起来就会报「未配置大模型」，AI 回复全线哑火。
const llmConfigFile = `E:\ai\data\llm.json`

var (
	llmOnce   sync.Once
	llmCached LLMConfig
)

// loadLLMConfig 先读环境变量，没有再读配置文件
func loadLLMConfig() LLMConfig {
	llmOnce.Do(func() {
		llmCached = LLMConfig{
			BaseURL: strings.TrimSpace(os.Getenv("RAG_CHAT_BASE_URL")),
			ApiKey:  strings.TrimSpace(os.Getenv("RAG_CHAT_API_KEY")),
			Model:   strings.TrimSpace(os.Getenv("RAG_CHAT_MODEL")),
		}

		if llmCached.BaseURL != "" && llmCached.ApiKey != "" {
			return
		}

		raw, err := os.ReadFile(llmConfigFile)
		if err != nil {
			return
		}

		// 文件可能带 UTF-8 BOM（编辑器/脚本写入），不去掉 json 会解析失败
		raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})

		var fromFile LLMConfig
		if err := json.Unmarshal(raw, &fromFile); err != nil {
			return
		}

		if llmCached.BaseURL == "" {
			llmCached.BaseURL = strings.TrimSpace(fromFile.BaseURL)
		}

		if llmCached.ApiKey == "" {
			llmCached.ApiKey = strings.TrimSpace(fromFile.ApiKey)
		}

		if llmCached.Model == "" {
			llmCached.Model = strings.TrimSpace(fromFile.Model)
		}
	})

	return llmCached
}

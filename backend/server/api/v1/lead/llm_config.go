package lead

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	leadSvc "github.com/flipped-aurora/gin-vue-admin/server/service/lead"
	"github.com/gin-gonic/gin"
)

type llmConfigPayload struct {
	BaseURL     string  `json:"baseUrl"`
	ApiKey      string  `json:"apiKey"`
	Model       string  `json:"model"`
	KBIDBuyer   string  `json:"kbIdBuyer"`
	KBIDSeller  string  `json:"kbIdSeller"`
	Temperature float64 `json:"temperature"`
}

// GetLLMConfig 读取大模型配置（key 打码）
func (leadApi *LeadApi) GetLLMConfig(c *gin.Context) {
	cfg := leadSvc.LoadLLMConfig()

	response.OkWithDetailed(gin.H{
		"baseUrl":     cfg.BaseURL,
		"apiKey":      maskKey(cfg.ApiKey),
		"hasKey":      cfg.ApiKey != "",
		"model":       cfg.Model,
		"kbIdBuyer":   cfg.KBIDBuyer,
		"kbIdSeller":  cfg.KBIDSeller,
		"temperature": cfg.Temperature,
		"filePath":    `E:\ai\data\llm.json`,
	}, "获取成功", c)
}

// SaveLLMConfig 保存大模型配置
func (leadApi *LeadApi) SaveLLMConfig(c *gin.Context) {
	var req llmConfigPayload

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	old := leadSvc.LoadLLMConfig()

	apiKey := req.ApiKey

	// 前端传回的是打码后的 key，说明没改，沿用旧值
	if apiKey == "" || contains(apiKey, "****") {
		apiKey = old.ApiKey
	}

	err := leadSvc.SaveLLMConfig(leadSvc.LLMConfig{
		BaseURL:     req.BaseURL,
		ApiKey:      apiKey,
		Model:       req.Model,
		KBIDBuyer:   req.KBIDBuyer,
		KBIDSeller:  req.KBIDSeller,
		Temperature: req.Temperature,
	})
	if err != nil {
		response.FailWithMessage("保存失败："+err.Error(), c)
		return
	}

	response.OkWithMessage("已保存，立即生效", c)
}

// TestLLMConfig 测试当前配置能否调通
func (leadApi *LeadApi) TestLLMConfig(c *gin.Context) {
	cfg := leadSvc.LoadLLMConfig()

	if cfg.ApiKey == "" {
		response.FailWithMessage("还没有配置 api_key", c)
		return
	}

	reply, err := leadSvc.TestLLM(c.Request.Context(), cfg)
	if err != nil {
		response.FailWithMessage("连接失败："+err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"reply": reply, "model": cfg.Model}, "连接正常", c)
}

func maskKey(key string) string {
	if key == "" {
		return ""
	}

	if len(key) <= 8 {
		return "****"
	}

	return key[:4] + "****" + key[len(key)-4:]
}

func contains(s, sub string) bool {
	return len(sub) == 0 || len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}

	return -1
}
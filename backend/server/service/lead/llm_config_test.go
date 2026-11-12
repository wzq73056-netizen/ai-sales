package lead

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLLMConfigFallback 验证配置文件能被读取
//
// 用临时文件 + LLM_CONFIG_FILE 环境变量，不依赖本机真实配置，
// 这样在 CI（Linux、没有 E:\ai\data）上也能跑。
func TestLLMConfigFallback(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "llm.json")

	content := `{
  "base_url": "https://example.com/v1",
  "api_key": "sk-test-key",
  "model": "test-model"
}`

	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("写入临时配置失败：%v", err)
	}

	t.Setenv("LLM_CONFIG_FILE", cfgPath)
	t.Setenv("RAG_CHAT_BASE_URL", "")
	t.Setenv("RAG_CHAT_API_KEY", "")
	t.Setenv("RAG_CHAT_MODEL", "")

	cfg := LoadLLMConfig()

	if cfg.BaseURL != "https://example.com/v1" {
		t.Fatalf("base_url 不符：%q", cfg.BaseURL)
	}

	if cfg.ApiKey != "sk-test-key" {
		t.Fatalf("api_key 不符：%q", cfg.ApiKey)
	}

	if cfg.Model != "test-model" {
		t.Fatalf("model 不符：%q", cfg.Model)
	}
}

// TestLLMConfigFileOverridesEnv 文件里的值应覆盖环境变量
func TestLLMConfigFileOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "llm.json")

	body := `{"base_url":"https://from-file/v1","api_key":"sk-file","model":"file-model"}`

	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatalf("写入临时配置失败：%v", err)
	}

	t.Setenv("LLM_CONFIG_FILE", cfgPath)
	t.Setenv("RAG_CHAT_BASE_URL", "https://from-env/v1")
	t.Setenv("RAG_CHAT_API_KEY", "sk-env")
	t.Setenv("RAG_CHAT_MODEL", "env-model")

	cfg := LoadLLMConfig()

	if cfg.BaseURL != "https://from-file/v1" {
		t.Fatalf("文件应覆盖环境变量，实际：%q", cfg.BaseURL)
	}
}

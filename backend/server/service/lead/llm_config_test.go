package lead

import "testing"

func TestLLMConfigFallback(t *testing.T) {
	cfg := loadLLMConfig()

	t.Logf("base=%q model=%q keyLen=%d", cfg.BaseURL, cfg.Model, len(cfg.ApiKey))

	if cfg.BaseURL == "" || cfg.ApiKey == "" {
		t.Fatalf("配置为空：base=%q keyLen=%d", cfg.BaseURL, len(cfg.ApiKey))
	}
}

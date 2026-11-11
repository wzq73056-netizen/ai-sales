package kb

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// KnowledgeStat 知识库统计（卡片上显示分片数）
type KnowledgeStat struct {
	KnowledgeId   int64 `json:"KnowledgeId"`
	FragmentCount int64 `json:"FragmentCount"`
}

// GetKnowledgeStats 按知识库聚合分片数量
func (kbService *KnowledgeService) GetKnowledgeStats(ctx context.Context) ([]KnowledgeStat, error) {
	var stats []KnowledgeStat

	err := global.GVA_DB.WithContext(ctx).
		Table("ai_knowledge_fragment").
		Select("knowledge_id, COUNT(*) AS fragment_count").
		Group("knowledge_id").
		Scan(&stats).Error

	return stats, err
}

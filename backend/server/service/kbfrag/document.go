package kbfrag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"gorm.io/gorm"
	kbfragModel "github.com/flipped-aurora/gin-vue-admin/server/model/kbfrag"
)

const chunkSize = 400
const chunkOverlap = 50

// GetDocumentList 分页取文档
func (kbfragService *KnowledgeFragmentService) GetDocumentList(ctx context.Context, knowledgeId int64, page, pageSize int) ([]kbfragModel.KnowledgeDocument, int64, error) {
	var list []kbfragModel.KnowledgeDocument
	var total int64

	db := global.GVA_DB.WithContext(ctx).Model(&kbfragModel.KnowledgeDocument{})
	if knowledgeId != 0 {
		db = db.Where("knowledge_id = ?", knowledgeId)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	err := db.Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&list).Error

	return list, total, err
}

// splitText 按字符切块，带重叠，尽量在换行处断开
func splitText(content string, size, overlap int) []string {
	runes := []rune(strings.TrimSpace(content))

	if len(runes) == 0 {
		return nil
	}

	var chunks []string

	for start := 0; start < len(runes); {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}

		// 尽量在换行/句号处断开
		if end < len(runes) {
			window := runes[end-60 : end]
			for i := len(window) - 1; i >= 0; i-- {
				if window[i] == '\n' || window[i] == '。' || window[i] == '.' {
					end = end - 60 + i + 1
					break
				}
			}
		}

		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		if end >= len(runes) {
			break
		}

		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// UploadDocument 保存文件 → 抽取文本 → 切块入库
func (kbfragService *KnowledgeFragmentService) UploadDocument(ctx context.Context, knowledgeId int64, filename string, content []byte) (kbfragModel.KnowledgeDocument, error) {
	now := time.Now()
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

	dir := filepath.Join("uploads", fmt.Sprintf("kb%d", knowledgeId))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return kbfragModel.KnowledgeDocument{}, err
	}

	saved := filepath.Join(dir, fmt.Sprintf("%d_%s", now.UnixMilli(), filename))
	if err := os.WriteFile(saved, content, 0o644); err != nil {
		return kbfragModel.KnowledgeDocument{}, err
	}

	doc := kbfragModel.KnowledgeDocument{
		KnowledgeId: knowledgeId,
		Name:        filename,
		Format:      ext,
		Size:        int64(len(content)),
		StoragePath: saved,
		Status:      0,
		CreatedAt:   now.Format("2006-01-02 15:04:05.000"),
		UpdatedAt:   now.Format("2006-01-02 15:04:05.000"),
	}

	if err := global.GVA_DB.WithContext(ctx).Create(&doc).Error; err != nil {
		return doc, err
	}

	// 目前只解析纯文本；pdf/docx 先存文件，状态留待后续接解析器
	if ext != "txt" && ext != "md" && ext != "csv" {
		return doc, nil
	}

	chunks := splitText(string(content), chunkSize, chunkOverlap)

	for i, chunk := range chunks {
		frag := kbfragModel.KnowledgeFragment{
			KnowledgeId: &knowledgeId,
			DocumentId:  &doc.Id,
			Idx:         int64Ptr(int64(i)),
			DocId:       stringPtr(fmt.Sprintf("%d", doc.Id)),
			Content:     stringPtr(chunk),
			CreatedAt:   stringPtr(now.Format("2006-01-02 15:04:05.000")),
			UpdatedAt:   stringPtr(now.Format("2006-01-02 15:04:05.000")),
		}
		if err := global.GVA_DB.WithContext(ctx).Create(&frag).Error; err != nil {
			return doc, err
		}
	}

	doc.ChunkCount = len(chunks)
	doc.Status = 1
	_ = global.GVA_DB.WithContext(ctx).Model(&doc).Updates(map[string]any{
		"chunk_count": len(chunks),
		"status":      1,
	}).Error

	return doc, nil
}

// DeleteDocument 删除文档及其片段
func (kbfragService *KnowledgeFragmentService) DeleteDocument(ctx context.Context, id int64) error {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("document_id = ?", id).Delete(&kbfragModel.KnowledgeFragment{}).Error; err != nil {
			return err
		}
		return tx.Delete(&kbfragModel.KnowledgeDocument{}, "id = ?", id).Error
	})
}

func int64Ptr(v int64) *int64    { return &v }
func stringPtr(v string) *string { return &v }

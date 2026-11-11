package lead

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)



// RecordDmMessages 监听器批量上报会话消息（按内容哈希去重）
func (leadService *LeadService) RecordDmMessages(platform string, deviceId int, items []leadModel.DmMessageInput) (int, error) {
	if platform == "" {
		return 0, fmt.Errorf("缺少 platform")
	}

	inserted := 0

	for _, item := range items {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}

		sum := sha1.Sum([]byte(platform + "|" + item.PeerId + "|" + text))
		hash := hex.EncodeToString(sum[:])

		row := map[string]any{
			"platform":     platform,
			"peer_name":    item.PeerName,
			"peer_id":      item.PeerId,
			"direction":    item.Direction,
			"content":      text,
			"content_hash": hash,
			"msg_time":     item.Time,
			"device_id":    deviceId,
		}

		res := global.GVA_DB.Table("ai_dm_message").Create(row)
		if res.Error == nil {
			inserted++

			// 对方发来消息 → 标记为「已回复」
			if item.Direction == 0 {
				leadService.MarkCampaignReplied(platform, item.PeerId, item.PeerName, text)
			}
		}
	}

	return inserted, nil
}

// DmConversation 会话摘要
type DmConversation struct {
	Platform string `json:"Platform"`
	PeerId   string `json:"PeerId"`
	PeerName string `json:"PeerName"`
	LastText string `json:"LastText"`
	LastTime string `json:"LastTime"`
	Direction int   `json:"Direction"`
	Count    int    `json:"Count"`
}

// GetDmConversations 按会话聚合
func (leadService *LeadService) GetDmConversations(platform string, limit int) []DmConversation {
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	type row struct {
		Platform  string
		PeerId    string
		PeerName  string
		Content   string
		MsgTime   string
		Direction int
	}

	var rows []row

	q := global.GVA_DB.Table("ai_dm_message").
		Select("platform, peer_id, peer_name, content, msg_time, direction").
		Order("id desc").Limit(400)

	if platform != "" {
		q = q.Where("platform = ?", platform)
	}

	q.Scan(&rows)

	seen := map[string]bool{}
	out := make([]DmConversation, 0, limit)

	for _, r := range rows {
		key := r.Platform + "|" + r.PeerId
		if seen[key] {
			continue
		}

		seen[key] = true

		out = append(out, DmConversation{
			Platform:  r.Platform,
			PeerId:    r.PeerId,
			PeerName:  r.PeerName,
			LastText:  r.Content,
			LastTime:  r.MsgTime,
			Direction: r.Direction,
			Count:     1,
		})

		if len(out) >= limit {
			break
		}
	}

	return out
}

// ===== 关键词库 =====



// GetScrapeKeywords 关键词列表
func (leadService *LeadService) GetScrapeKeywords(platform string) []leadModel.ScrapeKeyword {
	list := make([]leadModel.ScrapeKeyword, 0)

	q := global.GVA_DB.Table("ai_scrape_keyword").
		Select("id, platform, keyword, enabled, last_run_at").Order("id asc")

	if platform != "" {
		q = q.Where("platform = ?", platform)
	}

	q.Scan(&list)

	return list
}

// SaveScrapeKeyword 新增关键词
func (leadService *LeadService) SaveScrapeKeyword(item leadModel.ScrapeKeyword) error {
	if strings.TrimSpace(item.Keyword) == "" {
		return fmt.Errorf("关键词不能为空")
	}

	if item.Platform == "" {
		item.Platform = "xhs"
	}

	return global.GVA_DB.Exec(
		"INSERT IGNORE INTO ai_scrape_keyword (platform, keyword, enabled) VALUES (?, ?, ?)",
		item.Platform, strings.TrimSpace(item.Keyword), item.Enabled,
	).Error
}

// DeleteScrapeKeyword 删除关键词
func (leadService *LeadService) DeleteScrapeKeyword(id uint) error {
	return global.GVA_DB.Exec("DELETE FROM ai_scrape_keyword WHERE id = ?", id).Error
}

// ===== 黑名单 =====



// GetScrapeBlacklist 黑名单列表
func (leadService *LeadService) GetScrapeBlacklist() []leadModel.ScrapeBlacklist {
	list := make([]leadModel.ScrapeBlacklist, 0)
	global.GVA_DB.Table("ai_scrape_blacklist").Select("id, kind, value, reason").Order("id desc").Scan(&list)

	return list
}

// SaveScrapeBlacklist 新增黑名单
func (leadService *LeadService) SaveScrapeBlacklist(item leadModel.ScrapeBlacklist) error {
	if strings.TrimSpace(item.Value) == "" {
		return fmt.Errorf("内容不能为空")
	}

	if item.Kind == "" {
		item.Kind = "nickname"
	}

	return global.GVA_DB.Exec(
		"INSERT IGNORE INTO ai_scrape_blacklist (kind, value, reason) VALUES (?, ?, ?)",
		item.Kind, strings.TrimSpace(item.Value), item.Reason,
	).Error
}

// DeleteScrapeBlacklist 删除
func (leadService *LeadService) DeleteScrapeBlacklist(id uint) error {
	return global.GVA_DB.Exec("DELETE FROM ai_scrape_blacklist WHERE id = ?", id).Error
}

// ===== 导出 =====

// ExportLeads 导出线索为 xlsx，返回文件路径
func (leadService *LeadService) ExportLeads(source, grade string) (string, error) {
	var leads []leadModel.Lead

	q := global.GVA_DB.Model(&leadModel.Lead{}).Order("score desc, id desc").Limit(5000)

	if source != "" {
		q = q.Where("source = ?", source)
	}

	if grade != "" {
		q = q.Where("grade = ?", grade)
	}

	if err := q.Find(&leads).Error; err != nil {
		return "", err
	}

	return writeLeadXlsx(leads)
}



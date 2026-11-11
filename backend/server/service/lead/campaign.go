package lead

import (
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// ScriptStat 单个话术的效果
type ScriptStat struct {
	Script   string `json:"Script"`
	Platform string `json:"Platform"`
	Sent     int    `json:"Sent"`
	Replied  int    `json:"Replied"`
	Rate     float64 `json:"Rate"`
}

// ScriptStats 话术效果总览
type ScriptStats struct {
	Total   int          `json:"Total"`
	Replied int          `json:"Replied"`
	Rate    float64      `json:"Rate"`
	Items   []ScriptStat `json:"Items"`
	Recent  []CampaignRow `json:"Recent"`
}

// CampaignRow 一条发送记录
type CampaignRow struct {
	Id        uint   `json:"Id"`
	LeadId    uint   `json:"LeadId"`
	Platform  string `json:"Platform"`
	PeerName  string `json:"PeerName"`
	Script    string `json:"Script"`
	Status    int    `json:"Status"`
	ReplyText string `json:"ReplyText"`
	SentAt    string `json:"SentAt"`
	RepliedAt string `json:"RepliedAt"`
}

// RecordCampaign 发送成功后记一条，用于统计回复率
func (leadService *LeadService) RecordCampaign(leadId uint, platform, peerId, peerName, script string) {
	if strings.TrimSpace(peerName) == "" {
		return
	}

	global.GVA_DB.Table("ai_dm_campaign").Create(map[string]any{
		"lead_id":   leadId,
		"platform":  platform,
		"peer_id":   peerId,
		"peer_name": peerName,
		"script":    script,
		"status":    0,
		"sent_at":   time.Now(),
	})
}

// MarkCampaignReplied 对方回消息时标记
func (leadService *LeadService) MarkCampaignReplied(platform, peerId, peerName, text string) {
	// 优先用 peer_id，没有就按昵称匹配
	q := global.GVA_DB.Table("ai_dm_campaign").
		Where("platform = ? AND status = 0", platform)

	if peerId != "" {
		q = q.Where("peer_id = ?", peerId)
	} else {
		q = q.Where("peer_name = ?", peerName)
	}

	var row struct {
		Id uint
	}

	if err := q.Order("id desc").Limit(1).Scan(&row).Error; err != nil || row.Id == 0 {
		return
	}

	reply := text
	if len([]rune(reply)) > 200 {
		reply = string([]rune(reply)[:200])
	}

	global.GVA_DB.Table("ai_dm_campaign").Where("id = ?", row.Id).Updates(map[string]any{
		"status":     1,
		"reply_text": reply,
		"replied_at": time.Now(),
	})
}

// GetScriptStats 话术效果统计
func (leadService *LeadService) GetScriptStats() ScriptStats {
	stats := ScriptStats{Items: make([]ScriptStat, 0), Recent: make([]CampaignRow, 0)}

	var total, replied int64
	global.GVA_DB.Table("ai_dm_campaign").Count(&total)
	global.GVA_DB.Table("ai_dm_campaign").Where("status = 1").Count(&replied)

	stats.Total = int(total)
	stats.Replied = int(replied)

	if total > 0 {
		stats.Rate = float64(replied) / float64(total) * 100
	}

	// 按话术聚合（截断到 60 字做分组）
	type groupRow struct {
		Script   string
		Platform string
		Sent     int
		Replied  int
	}

	var groups []groupRow

	global.GVA_DB.Raw(`
		SELECT LEFT(script, 60) AS script,
		       MAX(platform) AS platform,
		       COUNT(*) AS sent,
		       SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) AS replied
		FROM ai_dm_campaign
		GROUP BY LEFT(script, 60)
		ORDER BY sent DESC
		LIMIT 30`).Scan(&groups)

	for _, g := range groups {
		item := ScriptStat{Script: g.Script, Platform: g.Platform, Sent: g.Sent, Replied: g.Replied}

		if g.Sent > 0 {
			item.Rate = float64(g.Replied) / float64(g.Sent) * 100
		}

		stats.Items = append(stats.Items, item)
	}

	var rows []CampaignRow

	global.GVA_DB.Table("ai_dm_campaign").
		Select("id, lead_id, platform, peer_name, script, status, reply_text, sent_at, replied_at").
		Order("id desc").Limit(20).Scan(&rows)

	for i := range rows {
		if len([]rune(rows[i].Script)) > 60 {
			rows[i].Script = string([]rune(rows[i].Script)[:60]) + "…"
		}

		if len([]rune(rows[i].ReplyText)) > 40 {
			rows[i].ReplyText = string([]rune(rows[i].ReplyText)[:40]) + "…"
		}
	}

	stats.Recent = rows

	return stats
}

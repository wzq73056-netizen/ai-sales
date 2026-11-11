package lead

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)



// GetXhsSchedule 读配置
func (leadService *LeadService) GetXhsSchedule() leadModel.XhsSchedule {
	var row leadModel.XhsSchedule

	global.GVA_DB.Table("ai_xhs_schedule").
		Select("id, keywords, notes, comment_limit, run_at, enabled, last_run_date").
		Order("id asc").Limit(1).Scan(&row)

	if row.Id == 0 {
		row = leadModel.XhsSchedule{Keywords: "鸡蛋批发", Notes: 3, CommentLimit: 30, RunAt: "09:00", Enabled: 0}
	}

	return row
}

// SaveXhsSchedule 保存配置
func (leadService *LeadService) SaveXhsSchedule(item leadModel.XhsSchedule) error {
	if item.Keywords == "" {
		item.Keywords = "鸡蛋批发"
	}

	if item.Notes <= 0 || item.Notes > 10 {
		item.Notes = 3
	}

	if item.CommentLimit <= 0 || item.CommentLimit > 100 {
		item.CommentLimit = 30
	}

	if item.RunAt == "" {
		item.RunAt = "09:00"
	}

	updates := map[string]any{
		"keywords":      item.Keywords,
		"notes":         item.Notes,
		"comment_limit": item.CommentLimit,
		"run_at":        item.RunAt,
		"enabled":       item.Enabled,
	}

	return global.GVA_DB.Table("ai_xhs_schedule").Where("id = ?", item.Id).Updates(updates).Error
}

// StartXhsScheduler 每分钟检查一次，到点就自动采集
func (leadService *LeadService) StartXhsScheduler() {
	go func() {
		ticker := time.NewTicker(time.Minute)

		for range ticker.C {
			cfg := leadService.GetXhsSchedule()

			if cfg.Enabled != 1 || cfg.RunAt == "" {
				continue
			}

			now := time.Now()
			today := now.Format("2006-01-02")
			hhmm := now.Format("15:04")

			if cfg.LastRunDate == today || hhmm < cfg.RunAt {
				continue
			}

			global.GVA_LOG.Info("小红书定时采集开始: " + cfg.Keywords)

			if _, err := leadService.StartXhsCollect(cfg.Keywords, cfg.Notes, cfg.CommentLimit); err != nil {
				global.GVA_LOG.Error("小红书定时采集失败: " + err.Error())
				continue
			}

			global.GVA_DB.Table("ai_xhs_schedule").Where("id = ?", cfg.Id).
				Update("last_run_date", today)
		}
	}()
}

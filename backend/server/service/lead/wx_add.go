package lead

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)



// WxAddQueueRow 队列一行
type WxAddQueueRow struct {
	Id          uint   `json:"Id"`
	LeadId      uint   `json:"LeadId"`
	Contact     string `json:"Contact"`
	ContactName string `json:"ContactName"`
	Remark      string `json:"Remark"`
	Status      int    `json:"Status"`
	Reason      string `json:"Reason"`
	ExecutedAt  string `json:"ExecutedAt"`
}

// GetWxAddConfig 读配置
func (leadService *LeadService) GetWxAddConfig() leadModel.WxAddConfig {
	var cfg leadModel.WxAddConfig

	global.GVA_DB.Table("ai_wx_add_config").
		Select("id, enabled, daily_limit, interval_min, interval_max, pause_minutes, window_start, window_end, remark, paused_until, last_run_at").
		Order("id asc").Limit(1).Scan(&cfg)

	return cfg
}

// SaveWxAddConfig 保存配置
func (leadService *LeadService) SaveWxAddConfig(cfg leadModel.WxAddConfig) error {
	if cfg.DailyLimit <= 0 || cfg.DailyLimit > 50 {
		cfg.DailyLimit = 15
	}

	if cfg.IntervalMin < 1 {
		cfg.IntervalMin = 10
	}

	if cfg.IntervalMax < cfg.IntervalMin {
		cfg.IntervalMax = cfg.IntervalMin + 5
	}

	if cfg.PauseMinutes <= 0 {
		cfg.PauseMinutes = 60
	}

	if cfg.WindowStart == "" {
		cfg.WindowStart = "09:00"
	}

	if cfg.WindowEnd == "" {
		cfg.WindowEnd = "21:00"
	}

	return global.GVA_DB.Table("ai_wx_add_config").Where("id = ?", cfg.Id).Updates(map[string]any{
		"enabled":       cfg.Enabled,
		"daily_limit":   cfg.DailyLimit,
		"interval_min":  cfg.IntervalMin,
		"interval_max":  cfg.IntervalMax,
		"pause_minutes": cfg.PauseMinutes,
		"window_start":  cfg.WindowStart,
		"window_end":    cfg.WindowEnd,
		"remark":        cfg.Remark,
	}).Error
}

// EnqueueWxAdd 把线索加入加好友队列（自动跳过没号码、黑名单、已在队列的）
func (leadService *LeadService) EnqueueWxAdd(leadIds []uint, remark string) (map[string]int, error) {
	result := map[string]int{"queued": 0, "skipped": 0, "noContact": 0}

	var blacklist []leadModel.ScrapeBlacklist
	global.GVA_DB.Table("ai_scrape_blacklist").Select("id, kind, value, reason").Scan(&blacklist)

	blackNames := map[string]bool{}
	for _, item := range blacklist {
		if item.Kind == "nickname" {
			blackNames[strings.ToLower(strings.TrimSpace(item.Value))] = true
		}
	}

	for _, id := range leadIds {
		var row leadModel.Lead

		if err := global.GVA_DB.Where("id = ?", id).First(&row).Error; err != nil {
			result["skipped"]++
			continue
		}

		contact := ""

		if row.Phone != nil && strings.TrimSpace(*row.Phone) != "" {
			contact = strings.TrimSpace(*row.Phone)
		} else if row.Wechat != nil && strings.TrimSpace(*row.Wechat) != "" {
			contact = strings.TrimSpace(*row.Wechat)
		}

		if contact == "" {
			result["noContact"]++
			continue
		}

		name := derefString(row.ContactName)

		if blackNames[strings.ToLower(name)] {
			result["skipped"]++
			continue
		}

		// 已成功或已排队的跳过
		var exist int64
		global.GVA_DB.Table("ai_wx_add_queue").
			Where("contact = ? AND status IN (0,1)", contact).Count(&exist)

		if exist > 0 {
			result["skipped"]++
			continue
		}

		global.GVA_DB.Table("ai_wx_add_queue").Create(map[string]any{
			"lead_id":      id,
			"contact":      contact,
			"contact_name": name,
			"remark":       remark,
			"status":       0,
		})

		result["queued"]++
	}

	return result, nil
}

// GetWxAddQueue 队列列表
func (leadService *LeadService) GetWxAddQueue(limit int) []WxAddQueueRow {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	list := make([]WxAddQueueRow, 0)

	global.GVA_DB.Table("ai_wx_add_queue").
		Select("id, lead_id, contact, contact_name, remark, status, reason, executed_at").
		Order("id desc").Limit(limit).Scan(&list)

	return list
}

// GetWxAddStats 今日执行情况
func (leadService *LeadService) GetWxAddStats() map[string]any {
	today := time.Now().Format("2006-01-02")

	type row struct {
		Status int
		Count  int
	}

	var rows []row

	global.GVA_DB.Raw(`SELECT status, COUNT(*) AS count FROM ai_wx_add_queue
		WHERE executed_at LIKE ? GROUP BY status`, today+"%").Scan(&rows)

	out := map[string]any{
		"todayOk": 0, "todayFail": 0, "todayTotal": 0,
		"pending": 0, "dailyLimit": leadService.GetWxAddConfig().DailyLimit,
	}

	for _, r := range rows {
		out["todayTotal"] = out["todayTotal"].(int) + r.Count

		if r.Status == 1 {
			out["todayOk"] = r.Count
		} else if r.Status >= 2 {
			out["todayFail"] = r.Count
		}
	}

	var pending int64
	global.GVA_DB.Table("ai_wx_add_queue").Where("status = 0").Count(&pending)
	out["pending"] = int(pending)

	return out
}

// RunWxAddOnce 手动触发：清掉间隔限制，等监听器几秒内领取执行
func (leadService *LeadService) RunWxAddOnce() (map[string]any, error) {
	cfg := leadService.GetWxAddConfig()

	if cfg.Enabled != 1 {
		return nil, fmt.Errorf("自动加好友未开启")
	}

	global.GVA_DB.Table("ai_wx_add_config").Where("id = ?", cfg.Id).
		Update("last_run_at", "")

	var pending int64
	global.GVA_DB.Table("ai_wx_add_queue").Where("status = 0").Count(&pending)

	return map[string]any{
		"pending": int(pending),
		"note":    "已放行，微信监听器会在几秒内执行",
	}, nil
}

// runWxAddDirect 直接执行（备用，后台进程拿不到桌面时用监听器）
func (leadService *LeadService) runWxAddDirect() (map[string]any, error) {
	cfg := leadService.GetWxAddConfig()

	if cfg.Enabled != 1 {
		return nil, fmt.Errorf("自动加好友未开启")
	}

	now := time.Now()

	if cfg.PausedUntil != "" {
		if until, err := time.ParseInLocation("2006-01-02 15:04:05", cfg.PausedUntil, time.Local); err == nil && now.Before(until) {
			return nil, fmt.Errorf("风控暂停中，%d 分钟后恢复", int(until.Sub(now).Minutes()))
		}
	}

	// 只在时间段内跑
	hhmm := now.Format("15:04")

	if hhmm < cfg.WindowStart || hhmm > cfg.WindowEnd {
		return nil, fmt.Errorf("不在执行时段（%s-%s）", cfg.WindowStart, cfg.WindowEnd)
	}

	// 今日上限
	stats := leadService.GetWxAddStats()

	if stats["todayTotal"].(int) >= cfg.DailyLimit {
		return nil, fmt.Errorf("今天已加 %d 个，达到上限", stats["todayTotal"])
	}

	// 间隔控制
	if cfg.LastRunAt != "" {
		if last, err := time.ParseInLocation("2006-01-02 15:04:05", cfg.LastRunAt, time.Local); err == nil {
			gap := int(now.Sub(last).Minutes())

			if gap < cfg.IntervalMin {
				return nil, fmt.Errorf("距上次 %d 分钟，还没到间隔 %d 分钟", gap, cfg.IntervalMin)
			}
		}
	}

	var job WxAddQueueRow

	if err := global.GVA_DB.Table("ai_wx_add_queue").
		Select("id, lead_id, contact, contact_name, remark, status").
		Where("status = 0").Order("id asc").Limit(1).Scan(&job).Error; err != nil || job.Id == 0 {
		return nil, fmt.Errorf("队列里没有待执行的")
	}

	remark := job.Remark
	if remark == "" {
		remark = cfg.Remark
	}

	state, raw := leadService.runAddFriendScript(job.Contact, remark, true)

	status := 2
	reason := state

	switch state {
	case "request_sent", "already_friend":
		status = 1
		reason = "已发送申请"
	case "can_add":
		status = 2
		reason = "找到可添加但没点确认（可能是脚本异常）"
	case "not_found":
		status = 3
		reason = "号码不存在"
	case "rate_limited":
		status = 4
		reason = "触发风控，暂停"
	default:
		status = 2
		reason = raw
	}

	global.GVA_DB.Table("ai_wx_add_queue").Where("id = ?", job.Id).Updates(map[string]any{
		"status":      status,
		"reason":      reason,
		"executed_at": now.Format("2006-01-02 15:04:05"),
	})

	global.GVA_DB.Table("ai_wx_add_config").Where("id = ?", cfg.Id).
		Update("last_run_at", now.Format("2006-01-02 15:04:05"))

	// 风控 → 暂停
	if state == "rate_limited" {
		until := now.Add(time.Duration(cfg.PauseMinutes) * time.Minute).Format("2006-01-02 15:04:05")
		global.GVA_DB.Table("ai_wx_add_config").Where("id = ?", cfg.Id).Update("paused_until", until)
	}

	return map[string]any{"leadId": job.LeadId, "contact": job.Contact, "state": state, "status": status}, nil
}

// runAddFriendScript 调 PowerShell 脚本
func (leadService *LeadService) runAddFriendScript(contact, remark string, send bool) (string, string) {
	args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", `E:\ai\scripts\wx_add_friend.ps1`, "-Contact", contact, "-Remark", remark}

	if send {
		args = append(args, "-Send")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", args...)
	cmd.Dir = `E:\ai\scripts`

	// Add-Type 需要可写的临时目录，否则报 Access is denied
	_ = os.MkdirAll(`E:\ai\data\tmp`, 0o755)
	cmd.Env = append(os.Environ(),
		`TEMP=E:\ai\data\tmp`,
		`TMP=E:\ai\data\tmp`,
	)

	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))

	// 调试：把原始输出写日志，便于排查后台启动和前台不一致的问题
	debugLine := fmt.Sprintf("[wx_add] contact=%s send=%v err=%v len=%d out=%.200s", contact, send, err, len(text), text)
	_ = os.WriteFile(`E:\ai\data\logs\wx_add_debug.log`, []byte(debugLine+"`n"), 0o644)

	if err != nil && text == "" {
		return "error", err.Error()
	}

	// 解析 json 里的 state
	state := "unknown"

	if idx := strings.Index(text, `"state":"`); idx >= 0 {
		rest := text[idx+len(`"state":"`):]

		if end := strings.Index(rest, `"`); end > 0 {
			state = rest[:end]
		}
	}

	return state, text
}



// ClaimWxAddJob 由监听器领取一条加好友任务（监听器进程在交互桌面，能操作微信）
func (leadService *LeadService) ClaimWxAddJob() (*WxAddQueueRow, string) {
	cfg := leadService.GetWxAddConfig()

	if cfg.Enabled != 1 {
		return nil, "自动加好友未开启"
	}

	now := time.Now()

	if cfg.PausedUntil != "" {
		if until, err := time.ParseInLocation("2006-01-02 15:04:05", cfg.PausedUntil, time.Local); err == nil && now.Before(until) {
			return nil, fmt.Sprintf("风控暂停中，%d 分钟后恢复", int(until.Sub(now).Minutes()))
		}
	}

	hhmm := now.Format("15:04")

	if hhmm < cfg.WindowStart || hhmm > cfg.WindowEnd {
		return nil, fmt.Sprintf("不在执行时段（%s-%s）", cfg.WindowStart, cfg.WindowEnd)
	}

	stats := leadService.GetWxAddStats()

	if stats["todayTotal"].(int) >= cfg.DailyLimit {
		return nil, fmt.Sprintf("今天已加 %d 个，达到上限", stats["todayTotal"])
	}

	if cfg.LastRunAt != "" {
		if last, err := time.ParseInLocation("2006-01-02 15:04:05", cfg.LastRunAt, time.Local); err == nil {
			if gap := int(now.Sub(last).Minutes()); gap < cfg.IntervalMin {
				return nil, fmt.Sprintf("距上次 %d 分钟，未到间隔 %d 分钟", gap, cfg.IntervalMin)
			}
		}
	}

	var job WxAddQueueRow

	global.GVA_DB.Table("ai_wx_add_queue").
		Select("id, lead_id, contact, contact_name, remark, status").
		Where("status = 0").Order("id asc").Limit(1).Scan(&job)

	if job.Id == 0 {
		return nil, "队列里没有待执行的"
	}

	remark := job.Remark
	if remark == "" {
		remark = cfg.Remark
	}

	job.Remark = remark

	global.GVA_DB.Table("ai_wx_add_queue").Where("id = ?", job.Id).Update("status", 5)
	global.GVA_DB.Table("ai_wx_add_config").Where("id = ?", cfg.Id).
		Update("last_run_at", now.Format("2006-01-02 15:04:05"))

	return &job, ""
}

// FinishWxAddJob 监听器回报执行结果
func (leadService *LeadService) FinishWxAddJob(id uint, state, detail string) error {
	cfg := leadService.GetWxAddConfig()
	now := time.Now()

	status := 2
	reason := state

	switch state {
	case "request_sent", "already_friend":
		status = 1
		reason = "已发送申请"
	case "not_found":
		status = 3
		reason = "号码不存在"
	case "rate_limited":
		status = 4
		reason = "触发风控，暂停"
	default:
		status = 2
		reason = detail
	}

	global.GVA_DB.Table("ai_wx_add_queue").Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"reason":      reason,
		"executed_at": now.Format("2006-01-02 15:04:05"),
	})

	if state == "rate_limited" {
		until := now.Add(time.Duration(cfg.PauseMinutes) * time.Minute).Format("2006-01-02 15:04:05")
		global.GVA_DB.Table("ai_wx_add_config").Where("id = ?", cfg.Id).Update("paused_until", until)
	}

	return nil
}
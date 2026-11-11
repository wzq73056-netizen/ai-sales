package lead

import (
	"os"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// DmLimit 私信限频配置
type DmLimit struct {
	DailyLimit  int64 `json:"DailyLimit"`
	IntervalSec int64 `json:"IntervalSec"`
}

// DmCheckResult 查重 + 限频结果
type DmCheckResult struct {
	Allowed     bool    `json:"Allowed"`
	Reason      string  `json:"Reason"`
	Message     string  `json:"Message"`
	LastSentAt  *string `json:"LastSentAt"`
	TodayCount  int64   `json:"TodayCount"`
	DailyLimit  int64   `json:"DailyLimit"`
	IntervalSec int64   `json:"IntervalSec"`
	WaitSec     int64   `json:"WaitSec"`
}

func envInt(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// GetDmLimit 读取限频配置（环境变量可覆盖）
func GetDmLimit() DmLimit {
	return DmLimit{
		DailyLimit:  envInt("DM_DAILY_LIMIT", 20),
		IntervalSec: envInt("DM_INTERVAL_SEC", 120),
	}
}

// lastSentAt 取最近一次成功发送时间（DATETIME 列直接扫进 *string 会得到 nil，必须格式化）
func lastSentAt(profileUrl string) (string, bool) {
	var sentAt string

	sqlStr := "SELECT DATE_FORMAT(sent_at, '%Y-%m-%d %H:%i:%s') FROM ai_douyin_dm WHERE status = 1"
	args := []any{}

	if profileUrl != "" {
		sqlStr += " AND profile_url = ?"
		args = append(args, profileUrl)
	}
	sqlStr += " ORDER BY id DESC LIMIT 1"

	row := global.GVA_DB.Raw(sqlStr, args...).Row()
	if row == nil {
		return "", false
	}
	if err := row.Scan(&sentAt); err != nil || sentAt == "" {
		return "", false
	}

	return sentAt, true
}

// CheckDouyinDm 发送前三重校验：同一人 / 每日上限 / 发送间隔
func (leadService *LeadService) CheckDouyinDm(profileUrl string) (DmCheckResult, error) {
	limit := GetDmLimit()
	res := DmCheckResult{Allowed: true, Reason: "ok", DailyLimit: limit.DailyLimit, IntervalSec: limit.IntervalSec}

	// 1) 同一人只发一次
	if sentAt, ok := lastSentAt(profileUrl); ok {
		res.Allowed = false
		res.Reason = "already_sent"
		res.Message = "已私信过该用户（" + sentAt + "）"
		res.LastSentAt = &sentAt
		return res, nil
	}

	// 2) 今日发送量
	today := time.Now().Format("2006-01-02")
	global.GVA_DB.Model(&leadModel.DouyinDm{}).
		Where("status = 1 AND sent_at >= ?", today+" 00:00:00").
		Count(&res.TodayCount)

	if res.TodayCount >= limit.DailyLimit {
		res.Allowed = false
		res.Reason = "daily_limit"
		res.Message = "已达每日发送上限 " + strconv.FormatInt(limit.DailyLimit, 10) + " 条"
		return res, nil
	}

	// 3) 发送间隔
	if sentAt, ok := lastSentAt(""); ok {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", sentAt, time.Local); err == nil {
			elapsed := int64(time.Since(t).Seconds())
			if elapsed < limit.IntervalSec {
				res.Allowed = false
				res.Reason = "interval"
				res.WaitSec = limit.IntervalSec - elapsed
				res.Message = "距上次发送不足间隔，还需等待 " + strconv.FormatInt(res.WaitSec, 10) + " 秒"
				res.LastSentAt = &sentAt
				return res, nil
			}
		}
	}

	return res, nil
}

// SaveDouyinDm 记录一次私信发送
func (leadService *LeadService) SaveDouyinDm(item leadModel.DouyinDm) (leadModel.DouyinDm, error) {
	now := time.Now().Format("2006-01-02 15:04:05")

	item.CreatedAt = &now
	item.UpdatedAt = &now

	if item.Status == 1 {
		item.SentAt = &now
	}

	if err := global.GVA_DB.Create(&item).Error; err != nil {
		return item, err
	}

	// 发送成功 → 线索池里同一个人自动标记为「已联系」
	if item.Status == 1 && item.ProfileUrl != "" {
		global.GVA_DB.Model(&leadModel.Lead{}).
			Where("source_url = ? AND status = 0", item.ProfileUrl).
			Updates(map[string]any{"status": 1, "updated_at": now})
	}

	if sentAt, ok := lastSentAt(item.ProfileUrl); ok {
		item.SentAt = &sentAt
	}

	return item, nil
}

// UpdateDmReply 记录对方回复
func (leadService *LeadService) UpdateDmReply(id int64, replyText string) error {
	now := time.Now().Format("2006-01-02 15:04:05")

	return global.GVA_DB.Model(&leadModel.DouyinDm{}).Where("id = ?", id).Updates(map[string]any{
		"reply_status": 1,
		"reply_text":   replyText,
		"reply_at":     now,
		"updated_at":   now,
	}).Error
}

// GetPendingDmList 待检查回复的私信（已发送且未回复，最近 14 天）
func (leadService *LeadService) GetPendingDmList(limit int) ([]leadModel.DouyinDm, error) {
	var list []leadModel.DouyinDm

	if limit <= 0 || limit > 50 {
		limit = 10
	}

	err := global.GVA_DB.
		Where("status = 1 AND reply_status = 0 AND sent_at >= DATE_SUB(NOW(), INTERVAL 14 DAY)").
		Order("id DESC").Limit(limit).Find(&list).Error

	return list, err
}

// GetDouyinDmList 私信记录列表
func (leadService *LeadService) GetDouyinDmList(page, pageSize int) ([]leadModel.DouyinDm, int64, error) {
	var list []leadModel.DouyinDm
	var total int64

	db := global.GVA_DB.Model(&leadModel.DouyinDm{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	err := db.Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&list).Error

	// DATETIME 列扫进 *string 是 nil，用格式化查询补齐
	for i := range list {
		var sentAt, createdAt, replyAt string
		global.GVA_DB.Raw(
			"SELECT COALESCE(DATE_FORMAT(sent_at,'%Y-%m-%d %H:%i:%s'),''), DATE_FORMAT(created_at,'%Y-%m-%d %H:%i:%s'), COALESCE(DATE_FORMAT(reply_at,'%Y-%m-%d %H:%i:%s'),'') FROM ai_douyin_dm WHERE id = ?",
			list[i].Id,
		).Row().Scan(&sentAt, &createdAt, &replyAt)

		if sentAt != "" {
			list[i].SentAt = &sentAt
		}
		if replyAt != "" {
			list[i].ReplyAt = &replyAt
		}
		list[i].CreatedAt = &createdAt
	}

	return list, total, err
}

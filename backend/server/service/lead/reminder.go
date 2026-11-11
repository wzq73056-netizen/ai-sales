package lead

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// ReminderRow 一条跟进提醒
type ReminderRow struct {
	Id         uint   `json:"Id"`
	NickName   string `json:"NickName"`
	Tier       string `json:"Tier"`
	Stage      string `json:"Stage"`
	Score      int    `json:"Score"`
	NextAction string `json:"NextAction"`
	NextFollow string `json:"NextFollow"`
	SilentDays int    `json:"SilentDays"`
	LastMsgAt  string `json:"LastMsgAt"`
	Bucket     string `json:"Bucket"` // overdue / today / week / silent
	Reason     string `json:"Reason"`
}

// ReminderBoard 提醒看板
type ReminderBoard struct {
	Overdue []ReminderRow `json:"Overdue"`
	Today   []ReminderRow `json:"Today"`
	Week    []ReminderRow `json:"Week"`
	Silent  []ReminderRow `json:"Silent"`
	Total   int           `json:"Total"`
}

// GetFollowupReminders 跟进提醒（逾期 / 今天 / 未来 7 天 / 沉睡）
func (leadService *LeadService) GetFollowupReminders() ReminderBoard {
	board := ReminderBoard{
		Overdue: make([]ReminderRow, 0),
		Today:   make([]ReminderRow, 0),
		Week:    make([]ReminderRow, 0),
		Silent:  make([]ReminderRow, 0),
	}

	type customerRow struct {
		Id         uint
		NickName   string
		Tier       string
		Stage      string
		Score      int
		NextAction string
		NextFollow string
		SilentDays int
		LastMsgAt  string
	}

	var rows []customerRow

	global.GVA_DB.Raw(`
		SELECT id, nick_name, tier, stage, COALESCE(score,0) AS score,
		       next_action, COALESCE(next_follow_at,'') AS next_follow,
		       COALESCE(silent_days,0) AS silent_days, COALESCE(last_msg_at,'') AS last_msg_at
		FROM ai_wx_customer
		ORDER BY next_follow_at ASC`).Scan(&rows)

	now := time.Now()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	weekEnd := todayEnd.AddDate(0, 0, 7)

	for _, r := range rows {
		item := ReminderRow{
			Id:         r.Id,
			NickName:   r.NickName,
			Tier:       r.Tier,
			Stage:      r.Stage,
			Score:      r.Score,
			NextAction: r.NextAction,
			NextFollow: r.NextFollow,
			SilentDays: r.SilentDays,
			LastMsgAt:  r.LastMsgAt,
		}

		if r.NextFollow != "" {
			if due, err := time.ParseInLocation("2006-01-02 15:04:05", r.NextFollow, time.Local); err == nil {
				switch {
				case due.Before(now):
					item.Bucket = "overdue"
					item.Reason = "跟进已逾期"
					board.Overdue = append(board.Overdue, item)
					continue
				case due.Before(todayEnd):
					item.Bucket = "today"
					item.Reason = "今天要跟进"
					board.Today = append(board.Today, item)
					continue
				case due.Before(weekEnd):
					item.Bucket = "week"
					item.Reason = "未来 7 天内"
					board.Week = append(board.Week, item)
					continue
				}
			}
		}

		// 没有安排时间，但沉默太久 → 沉睡客户
		if r.SilentDays >= 7 {
			item.Bucket = "silent"
			item.Reason = "超过 7 天没说话"
			board.Silent = append(board.Silent, item)
		}
	}

	board.Total = len(board.Overdue) + len(board.Today) + len(board.Week) + len(board.Silent)

	return board
}

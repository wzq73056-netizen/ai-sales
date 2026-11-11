package lead

import (
	"sort"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// InboxItem 统一收件箱的一条
type InboxItem struct {
	Kind    string `json:"kind"`    // handoff 待接管 / message 最近消息
	Channel string `json:"channel"` // wechat / douyin / xhs
	Id      uint   `json:"Id"`
	Name    string `json:"Name"`
	Text    string `json:"Text"`
	Time    string `json:"Time"`
	Urgency int    `json:"Urgency"`
	Status  int    `json:"Status"`
	Tag     string `json:"Tag"`
}

// InboxStats 收件箱统计
type InboxStats struct {
	HandoffCount int `json:"handoffCount"`
	MessageCount int `json:"messageCount"`
	WechatCount  int `json:"wechatCount"`
	DouyinCount  int `json:"douyinCount"`
	XhsCount     int `json:"xhsCount"`
}

// Inbox 收件箱
type Inbox struct {
	Stats InboxStats  `json:"stats"`
	Items []InboxItem `json:"items"`
}

// GetInbox 聚合三个平台的待接管 + 最近消息
func (leadService *LeadService) GetInbox(channel string, limit int) (Inbox, error) {
	if limit <= 0 || limit > 200 {
		limit = 60
	}

	result := Inbox{Items: make([]InboxItem, 0, limit)}

	// 1) 待接管工单（微信 + 抖音）
	var handoffs []leadModel.WxHandoff

	query := global.GVA_DB.Where("status = 0").Order("urgency desc, id desc").Limit(50)
	if channel != "" {
		query = query.Where("channel = ?", channel)
	}

	query.Find(&handoffs)

	for _, row := range handoffs {
		timeText := ""
		if row.CreatedAt != nil {
			timeText = *row.CreatedAt
			if len(timeText) > 16 {
				timeText = timeText[:16]
			}
		}

		result.Items = append(result.Items, InboxItem{
			Kind:    "handoff",
			Channel: row.Channel,
			Id:      uint(row.Id),
			Name:    row.NickName,
			Text:    row.LastMsg,
			Time:    timeText,
			Urgency: row.Urgency,
			Status:  row.Status,
			Tag:     row.Category,
		})
	}

	result.Stats.HandoffCount = len(handoffs)

	// 2) 微信客户最近消息
	if channel == "" || channel == "wechat" {
		var customers []leadModel.WxCustomer

		global.GVA_DB.Order("last_msg_at desc, id desc").Limit(30).Find(&customers)

		for _, row := range customers {
			timeText := ""
			if row.LastMsgAt != nil {
				timeText = *row.LastMsgAt
				if len(timeText) > 16 {
					timeText = timeText[:16]
				}
			}

			stage := row.Stage
			if stage == "" {
				stage = row.Tier + " 类"
			}

			result.Items = append(result.Items, InboxItem{
				Kind:    "message",
				Channel: "wechat",
				Id:      uint(row.Id),
				Name:    row.NickName,
				Text:    "最近对话 · " + stage,
				Time:    timeText,
				Urgency: 0,
				Status:  row.FollowStatus,
				Tag:     stage,
			})
			result.Stats.WechatCount++
		}
	}

	// 3) 抖音 / 小红书：来自私信消息表的会话
	for _, platform := range []string{"douyin", "xhs"} {
		if channel != "" && channel != platform {
			continue
		}

		convs := leadService.GetDmConversations(platform, 25)

		for _, conv := range convs {
			result.Items = append(result.Items, InboxItem{
				Kind:    "message",
				Channel: platform,
				Id:      0,
				Name:    conv.PeerName,
				Text:    conv.LastText,
				Time:    conv.LastTime,
				Urgency: 0,
				Status:  conv.Direction,
				Tag:     "私信",
			})

			if platform == "douyin" {
				result.Stats.DouyinCount++
			} else {
				result.Stats.XhsCount++
			}
		}
	}
	result.Stats.MessageCount = result.Stats.WechatCount + result.Stats.DouyinCount + result.Stats.XhsCount

	// 待接管优先，其次按时间倒序
	sort.SliceStable(result.Items, func(i, j int) bool {
		a, b := result.Items[i], result.Items[j]

		if (a.Kind == "handoff") != (b.Kind == "handoff") {
			return a.Kind == "handoff"
		}

		return a.Time > b.Time
	})

	if len(result.Items) > limit {
		result.Items = result.Items[:limit]
	}

	return result, nil
}

var _ = time.Now

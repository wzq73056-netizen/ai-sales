package lead

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// WxChatSnapshot 采集端上报的会话快照
type WxChatSnapshot struct {
	ChatId     string   `json:"ChatId"`
	NickName   string   `json:"NickName"`
	LastMsgAt  string   `json:"LastMsgAt"`
	LastFromMe bool     `json:"LastFromMe"`
	MyCount    int      `json:"MyCount"`
	TheirCount int      `json:"TheirCount"`
	Texts      []string `json:"Texts"`
}

var (
	askPriceWords   = []string{"多少钱", "什么价", "价格", "报价", "几块", "一斤", "怎么卖", "啥价"}
	askVolumeWords  = []string{"多少件", "起订", "起送", "要多少", "一天要", "每天要", "量"}
	askFreightWords = []string{"运费", "到货", "发货", "包邮", "送到", "多久到"}
	askSettleWords  = []string{"结算", "现结", "月结", "账期", "打款", "开票"}
	contactWords    = []string{"电话", "微信", "手机号", "联系"}
	peerWords       = []string{"批发", "供应", "厂家", "养殖场", "贸易", "商行", "公司"}
)

func containsAny(text string, words []string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}

	return false
}

// scoreChat 规则打分，返回分数、分层、标签、依据
func scoreChat(item WxChatSnapshot, silentDays int) (int, string, []string, string) {
	joined := strings.Join(item.Texts, " ")
	score := 20
	tags := []string{}
	reasons := []string{"基础分20"}

	if containsAny(joined, askPriceWords) {
		score += 25
		tags = append(tags, "问过价")
		reasons = append(reasons, "问过价+25")
	}
	if containsAny(joined, askVolumeWords) {
		score += 15
		tags = append(tags, "问过量")
		reasons = append(reasons, "问过量+15")
	}
	if containsAny(joined, askFreightWords) {
		score += 15
		tags = append(tags, "问过运费")
		reasons = append(reasons, "问过运费+15")
	}
	if containsAny(joined, askSettleWords) {
		score += 10
		tags = append(tags, "谈结算")
		reasons = append(reasons, "谈过结算+10")
	}
	if containsAny(joined, contactWords) {
		score += 20
		tags = append(tags, "给过联系方式")
		reasons = append(reasons, "涉及联系方式+20")
	}

	if item.TheirCount >= 5 {
		score += 10
	}
	if item.TheirCount >= 10 {
		score += 5
	}
	if !item.LastFromMe {
		score += 10
		reasons = append(reasons, "对方在等回复+10")
	}

	if silentDays > 7 {
		score -= 15
		reasons = append(reasons, "沉默超过7天-15")
	}
	if silentDays > 14 {
		tags = append(tags, "冷客户")
		score -= 10
		reasons = append(reasons, "沉默超过14天-10")
	}

	if containsAny(item.NickName, peerWords) {
		score -= 20
		tags = append(tags, "疑似同行")
		reasons = append(reasons, "名称疑似同行-20")
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	tier := "D"
	switch {
	case score >= 70:
		tier = "A"
	case score >= 50:
		tier = "B"
	case score >= 30:
		tier = "C"
	}

	return score, tier, tags, strings.Join(reasons, "，")
}

// SyncWxCustomers 批量同步客户价值（采集端每小时/每次变化上报）
func (leadService *LeadService) SyncWxCustomers(ctx context.Context, items []WxChatSnapshot) (int, error) {
	now := time.Now()
	nowText := now.Format("2006-01-02 15:04:05")
	count := 0

	for _, item := range items {
		if item.ChatId == "" {
			continue
		}

		silentDays := 0
		if item.LastMsgAt != "" {
			if t, err := time.ParseInLocation("2006-01-02 15:04:05", item.LastMsgAt, time.Local); err == nil {
				silentDays = int(now.Sub(t).Hours() / 24)
			}
		}

		score, tier, tags, reason := scoreChat(item, silentDays)

		// 对话有变化时用大模型再评一次（规则分 40% + 模型分 60%）
		stage, llmReason, nextAction := "", "", ""
		llmScore := -1

		// chat_id 每次解密都变，用昵称当唯一键，避免同一客户反复建行
		var existing leadModel.WxCustomer
		hasExisting := global.GVA_DB.Where("nick_name = ?", item.NickName).First(&existing).Error == nil

		changed := !hasExisting || existing.MyCount != item.MyCount || existing.TheirCount != item.TheirCount || existing.Stage == ""

		if changed && len(item.Texts) > 0 {
			history := strings.Join(item.Texts, "\n")
			if len([]rune(history)) > 1500 {
				history = string([]rune(history)[:1500])
			}

			result, err := leadService.ScoreCustomerByLLM(ctx, item.NickName, history)
			if err != nil {
				global.GVA_LOG.Error("客户评分失败", zap.Error(err))
			} else {
				llmScore = result.Score
				score = MergeScore(score, result.Score)
				stage = result.Stage
				llmReason = result.Reason
				nextAction = result.NextAction
				tags = append(tags, result.Tags...)

				switch {
				case score >= 70:
					tier = "A"
				case score >= 50:
					tier = "B"
				case score >= 30:
					tier = "C"
				default:
					tier = "D"
				}
			}
		}

		// 建议跟进时间
		next := now.AddDate(0, 0, 3)
		switch tier {
		case "A":
			next = now
		case "B":
			next = now.AddDate(0, 0, 3)
		case "C":
			next = now.AddDate(0, 0, 7)
		default:
			next = now.AddDate(0, 0, 30)
		}

		lastFromMe := 0
		if item.LastFromMe {
			lastFromMe = 1
		}

		// 用昵称做唯一键：chat_id 每轮解密都会变
		var exist leadModel.WxCustomer
		err := global.GVA_DB.Where("nick_name = ?", item.NickName).First(&exist).Error

		values := map[string]any{
			"nick_name":      item.NickName,
			// chat_id 每轮解密都会变，跟着刷新，否则客服页面的跟进会挂到旧 id 上
			"chat_id":        item.ChatId,
			"stage":          stage,
			"llm_score":      llmScore,
			"llm_reason":     llmReason,
			"next_action":    nextAction,
			"score":          score,
			"tier":           tier,
			"tags":           strings.Join(tags, ","),
			"my_count":       item.MyCount,
			"their_count":    item.TheirCount,
			"last_msg_at":    item.LastMsgAt,
			"last_from_me":   lastFromMe,
			"silent_days":    silentDays,
			"next_follow_at": next.Format("2006-01-02 15:04:05"),
			"reason":         reason,
			"updated_at":     nowText,
		}

		if err == nil {
			if e := global.GVA_DB.Model(&exist).Updates(values).Error; e == nil {
				count++
			}
			continue
		}

		values["chat_id"] = item.ChatId
		values["created_at"] = nowText

		if e := global.GVA_DB.Model(&leadModel.WxCustomer{}).Create(values).Error; e == nil {
			count++
		}
	}

	return count, nil
}

// GetWxCustomerList 客户跟进列表
func (leadService *LeadService) GetWxCustomerList(page, pageSize, minScore int, tier, stage, keyword, side, sortBy, order string) ([]leadModel.WxCustomer, int64, error) {
	var list []leadModel.WxCustomer
	var total int64

	db := global.GVA_DB.Model(&leadModel.WxCustomer{})

	if minScore > 0 {
		db = db.Where("score >= ?", minScore)
	}

	if tier != "" {
		db = db.Where("tier = ?", tier)
	}

	if stage != "" {
		db = db.Where("stage = ?", stage)
	}

	if keyword != "" {
		db = db.Where("nick_name LIKE ?", "%"+keyword+"%")
	}

	// 方向筛选：buyer=客户 / supplier=供货商
	if side != "" {
		db = db.Where("side = ?", side)
	}

	// 排序白名单（空值排最后，避免沉默天数为 NULL 时跑最前）
	sortMap := map[string]string{
		"nick_name":      "nick_name",
		"NickName":       "nick_name",
		"score":          "score",
		"Score":          "score",
		"next_follow_at": "next_follow_at",
		"NextFollowAt":   "next_follow_at",
		"last_msg_at":    "last_msg_at",
		"LastMsgAt":      "last_msg_at",
		"silent_days":    "silent_days",
		"SilentDays":     "silent_days",
	}

	if col, ok := sortMap[sortBy]; ok {
		dir := "ASC"
		if order == "descending" || order == "desc" {
			dir = "DESC"
		}

		db = db.Order(col + " IS NULL, " + col + " " + dir)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	err := db.Order("score DESC, id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&list).Error

	for i := range list {
		var lastMsg, nextAt, created string

		global.GVA_DB.Raw(
			"SELECT COALESCE(DATE_FORMAT(last_msg_at,'%Y-%m-%d %H:%i'),''), COALESCE(DATE_FORMAT(next_follow_at,'%Y-%m-%d %H:%i'),''), DATE_FORMAT(created_at,'%Y-%m-%d %H:%i') FROM ai_wx_customer WHERE id = ?",
			list[i].Id,
		).Row().Scan(&lastMsg, &nextAt, &created)

		if lastMsg != "" {
			list[i].LastMsgAt = &lastMsg
		}
		if nextAt != "" {
			list[i].NextFollowAt = &nextAt
		}
		list[i].CreatedAt = &created
	}

	return list, total, err
}

// MarkWxCustomerFollow 标记跟进状态
func (leadService *LeadService) MarkWxCustomerFollow(id int64, status int) error {
	return global.GVA_DB.Model(&leadModel.WxCustomer{}).Where("id = ?", id).
		Update("follow_status", status).Error
}

// UpdateWxCustomer 更新客户关键字段（阶段 / 下次跟进 / 下一步 / 标签）
func (leadService *LeadService) UpdateWxCustomer(id uint, stage, nextFollowAt, nextAction, tags string) error {
	updates := map[string]any{}

	if stage != "" {
		updates["stage"] = stage
	}

	if nextFollowAt != "" {
		updates["next_follow_at"] = nextFollowAt
	}

	if nextAction != "" {
		updates["next_action"] = nextAction
	}

	if tags != "" {
		updates["tags"] = tags
	}

	if len(updates) == 0 {
		return nil
	}

	return global.GVA_DB.Table("ai_wx_customer").Where("id = ?", id).Updates(updates).Error
}
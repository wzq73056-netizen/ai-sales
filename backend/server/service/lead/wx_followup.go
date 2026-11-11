package lead

import (
	"sort"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// AddWxFollowup 记一条跟进
func (leadService *LeadService) AddWxFollowup(nick, chatId, kind, content, mediaUrl, operator string) error {
	if operator == "" {
		operator = "system"
	}

	now := time.Now()
	record := leadModel.WxFollowup{
		NickName:  nick,
		ChatId:    chatId,
		Kind:      kind,
		Content:   EncryptContent(content),
		MediaUrl:  mediaUrl,
		Operator:  operator,
		CreatedAt: &now,
	}

	return global.GVA_DB.Create(&record).Error
}

// GetWxFollowupList 某客户的跟进记录（新的在前）
func (leadService *LeadService) GetWxFollowupList(nick string, limit int) ([]leadModel.WxFollowup, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	var records []leadModel.WxFollowup

	err := global.GVA_DB.Where("nick_name = ?", nick).Order("id DESC").Limit(limit).Find(&records).Error

	for index := range records {
		records[index].Content = DecryptContent(records[index].Content)
	}

	return records, err
}

// GetWxCustomerGrouped 按客户合并（同一个昵称只留一条，取最高分那条的信息）
func (leadService *LeadService) GetWxCustomerGrouped(minScore, limit int) ([]leadModel.WxCustomer, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	var rows []leadModel.WxCustomer

	err := global.GVA_DB.Order("score DESC, id DESC").Limit(limit * 5).Find(&rows).Error
	if err != nil {
		return nil, err
	}

	merged := make(map[string]leadModel.WxCustomer)
	counts := make(map[string]int)

	for _, row := range rows {
		key := row.NickName
		if key == "" {
			key = row.ChatId
		}

		counts[key]++

		best, ok := merged[key]

		// 优先保留有模型评估（阶段非空）的那条，其次分高、再其次新
		if !ok {
			merged[key] = row
			continue
		}

		bestScored := best.Stage != ""
		rowScored := row.Stage != ""

		better := false
		switch {
		case rowScored && !bestScored:
			better = true
		case rowScored == bestScored && row.Score > best.Score:
			better = true
		case rowScored == bestScored && row.Score == best.Score && row.Id > best.Id:
			better = true
		}

		if better {
			row.MyCount = max(best.MyCount, row.MyCount)
			row.TheirCount = max(best.TheirCount, row.TheirCount)
			row.SilentDays = min(best.SilentDays, row.SilentDays)
			merged[key] = row
		} else {
			best.MyCount = max(best.MyCount, row.MyCount)
			best.TheirCount = max(best.TheirCount, row.TheirCount)
			if row.SilentDays < best.SilentDays {
				best.SilentDays = row.SilentDays
			}
			merged[key] = best
		}
	}

	result := make([]leadModel.WxCustomer, 0, len(merged))

	for _, row := range merged {
		if row.Score >= minScore {
			result = append(result, row)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// GetWxCustomerDetail 客户详情
func (leadService *LeadService) GetWxCustomerDetail(nick string) (leadModel.WxCustomerDetail, error) {
	var customer leadModel.WxCustomer

	err := global.GVA_DB.Where("nick_name = ?", nick).Order("score DESC, id DESC").First(&customer).Error
	if err != nil {
		return leadModel.WxCustomerDetail{}, err
	}

	followups, _ := leadService.GetWxFollowupList(nick, 100)

	return leadModel.WxCustomerDetail{Customer: customer, Followups: followups}, nil
}

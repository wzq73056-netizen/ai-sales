package lead

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// FunnelStage 漏斗一层
type FunnelStage struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Rate  float64 `json:"rate"`
}

// FunnelCategory 工单类别分布
type FunnelCategory struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// FunnelDay 每日新增
type FunnelDay struct {
	Date      string `json:"date"`
	Customers int    `json:"customers"`
	Handoffs  int    `json:"handoffs"`
}

// FunnelStats 转化漏斗
type FunnelStats struct {
	Total        int              `json:"total"`
	Stages       []FunnelStage    `json:"stages"`
	Categories   []FunnelCategory `json:"categories"`
	Trend        []FunnelDay      `json:"trend"`
	DealCount    int              `json:"dealCount"`
	DealRate     float64          `json:"dealRate"`
	AvgScore     int              `json:"avgScore"`
	HandoffTotal int              `json:"handoffTotal"`
}

// stageOrder 漏斗阶段顺序
var stageOrder = []string{"了解", "比价", "议价", "待成交", "已成交"}

// GetFunnelStats 统计转化漏斗
func (leadService *LeadService) GetFunnelStats() (FunnelStats, error) {
	stats := FunnelStats{
		Stages:     make([]FunnelStage, 0, 8),
		Categories: make([]FunnelCategory, 0, 8),
		Trend:      make([]FunnelDay, 0, 7),
	}

	var customers []leadModel.WxCustomer

	if err := global.GVA_DB.Find(&customers).Error; err != nil {
		return stats, err
	}

	// 同一昵称只算一次
	seen := map[string]leadModel.WxCustomer{}

	for _, row := range customers {
		key := row.NickName
		if key == "" {
			key = row.ChatId
		}

		if old, ok := seen[key]; !ok || row.Score > old.Score {
			seen[key] = row
		}
	}

	stats.Total = len(seen)

	stageCount := map[string]int{}
	scoreSum := 0

	for _, row := range seen {
		stage := row.Stage
		if stage == "" {
			stage = "了解"
		}

		stageCount[stage]++
		scoreSum += row.Score

		if row.Stage == "已成交" || row.FollowStatus == 2 {
			stats.DealCount++
		}
	}

	if stats.Total > 0 {
		stats.AvgScore = scoreSum / stats.Total
		stats.DealRate = float64(stats.DealCount) / float64(stats.Total) * 100
	}

	// 漏斗：客户当前所处阶段越深，说明它已经过前面所有阶段
	for index, name := range stageOrder {
		reached := 0

		for deeper := index; deeper < len(stageOrder); deeper++ {
			reached += stageCount[stageOrder[deeper]]
		}

		rate := 0.0
		if stats.Total > 0 {
			rate = float64(reached) / float64(stats.Total) * 100
		}

		stats.Stages = append(stats.Stages, FunnelStage{Name: name, Count: reached, Rate: rate})
	}

	// 工单类别分布
	var categories []FunnelCategory

	global.GVA_DB.Model(&leadModel.WxHandoff{}).
		Select("category as name, count(*) as count").
		Group("category").
		Order("count desc").
		Scan(&categories)

	// 过滤空类别（保持非 nil，前端才好判断）
	filtered := make([]FunnelCategory, 0, len(categories))

	for _, item := range categories {
		if item.Name != "" && item.Count > 0 {
			filtered = append(filtered, item)
		}
	}

	stats.Categories = filtered

	var handoffTotal int64
	global.GVA_DB.Model(&leadModel.WxHandoff{}).Count(&handoffTotal)
	stats.HandoffTotal = int(handoffTotal)

	// 最近 7 天趋势
	for day := 6; day >= 0; day-- {
		date := time.Now().AddDate(0, 0, -day).Format("2006-01-02")

		var customerCount int64
		var handoffCount int64

		global.GVA_DB.Model(&leadModel.WxCustomer{}).
			Where("DATE(created_at) = ?", date).Count(&customerCount)

		global.GVA_DB.Model(&leadModel.WxHandoff{}).
			Where("DATE(created_at) = ?", date).Count(&handoffCount)

		stats.Trend = append(stats.Trend, FunnelDay{
			Date:      date[5:],
			Customers: int(customerCount),
			Handoffs:  int(handoffCount),
		})
	}

	return stats, nil
}

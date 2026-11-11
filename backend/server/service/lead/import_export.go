package lead

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// ConvertLeads 批量把线索转成客户
func (leadService *LeadService) ConvertLeads(ids []uint) map[string]int {
	result := map[string]int{"converted": 0, "duplicated": 0, "failed": 0}

	for _, id := range ids {
		data, err := leadService.ConvertLeadToCustomer(id)
		if err != nil {
			result["failed"]++
			continue
		}

		if dup, ok := data["duplicated"].(bool); ok && dup {
			result["duplicated"]++
			continue
		}

		result["converted"]++
	}

	return result
}

// LeadSourceStat 来源分布
type LeadSourceStat struct {
	Source string `json:"Source"`
	Total  int    `json:"Total"`
	GradeA int    `json:"GradeA"`
	GradeB int    `json:"GradeB"`
	GradeC int    `json:"GradeC"`
	WithContact int `json:"WithContact"`
}

// FollowupHealth 跟进健康度
type FollowupHealth struct {
	CustomerTotal  int     `json:"CustomerTotal"`
	Overdue        int     `json:"Overdue"`
	OverdueRate    float64 `json:"OverdueRate"`
	Silent7        int     `json:"Silent7"`
	Silent30       int     `json:"Silent30"`
	AvgSilentDays  float64 `json:"AvgSilentDays"`
	DealCount      int     `json:"DealCount"`
	ValidCount     int     `json:"ValidCount"`
}

// GetLeadAnalytics 报表：来源分布 + 跟进健康度
func (leadService *LeadService) GetLeadAnalytics() map[string]any {
	sources := make([]LeadSourceStat, 0)

	global.GVA_DB.Raw(`
		SELECT COALESCE(source,'其他') AS source,
		       COUNT(*) AS total,
		       SUM(CASE WHEN grade IN ('S','A') OR grade IN ('S1','A1','S2','A2') THEN 1 ELSE 0 END) AS grade_a,
		       SUM(CASE WHEN grade IN ('B','B1','B2') THEN 1 ELSE 0 END) AS grade_b,
		       SUM(CASE WHEN grade IN ('C','C1','C2','D') THEN 1 ELSE 0 END) AS grade_c,
		       SUM(CASE WHEN (phone IS NOT NULL AND phone <> '') OR (wechat IS NOT NULL AND wechat <> '') THEN 1 ELSE 0 END) AS with_contact
		FROM ai_lead GROUP BY COALESCE(source,'其他') ORDER BY total DESC`).Scan(&sources)

	// 方向分布 + 每个方向的转化：线索 → A 类 → 有联系方式 → 转客户 → 成交
	sideRows := make([]struct {
		Side        string  `json:"side"`
		Total       int64   `json:"total"`
		GradeA      int64   `json:"grade_a"`
		WithContact int64   `json:"with_contact"`
		Converted   int64   `json:"converted"`
		Deals       int64   `json:"deals"`
		AvgScore    float64 `json:"avg_score"`
	}, 0)

	global.GVA_DB.Raw(`
		SELECT COALESCE(NULLIF(side,''),'buyer') AS side,
		       COUNT(*) AS total,
		       SUM(CASE WHEN grade IN ('S','A','S1','A1','S2','A2') THEN 1 ELSE 0 END) AS grade_a,
		       SUM(CASE WHEN (phone IS NOT NULL AND phone <> '') OR (wechat IS NOT NULL AND wechat <> '') THEN 1 ELSE 0 END) AS with_contact,
		       SUM(CASE WHEN status >= 2 THEN 1 ELSE 0 END) AS converted,
		       SUM(CASE WHEN status = 3 THEN 1 ELSE 0 END) AS deals,
		       COALESCE(AVG(score), 0) AS avg_score
		FROM ai_lead
		GROUP BY COALESCE(NULLIF(side,''),'buyer')`).Scan(&sideRows)

	sideStats := make([]map[string]any, 0, len(sideRows))

	for _, row := range sideRows {
		name := "我方客户"
		if row.Side == "supplier" {
			name = "供我方"
		}

		convertRate := 0.0
		if row.Total > 0 {
			convertRate = float64(row.Converted) / float64(row.Total) * 100
		}

		sideStats = append(sideStats, map[string]any{
			"Side":        row.Side,
			"Name":        name,
			"Total":       row.Total,
			"GradeA":      row.GradeA,
			"WithContact": row.WithContact,
			"Converted":   row.Converted,
			"Deals":       row.Deals,
			"AvgScore":    row.AvgScore,
			"ConvertRate": convertRate,
		})
	}

	health := FollowupHealth{}

	var total int64
	global.GVA_DB.Table("ai_wx_customer").Count(&total)
	health.CustomerTotal = int(total)

	var overdue int64
	global.GVA_DB.Table("ai_wx_customer").
		Where("next_follow_at IS NOT NULL AND next_follow_at <> '' AND next_follow_at < NOW()").Count(&overdue)
	health.Overdue = int(overdue)

	if health.CustomerTotal > 0 {
		health.OverdueRate = float64(overdue) / float64(health.CustomerTotal) * 100
	}

	var silent7, silent30 int64
	global.GVA_DB.Table("ai_wx_customer").Where("silent_days >= 7").Count(&silent7)
	global.GVA_DB.Table("ai_wx_customer").Where("silent_days >= 30").Count(&silent30)
	health.Silent7 = int(silent7)
	health.Silent30 = int(silent30)

	var avgSilent *float64
	global.GVA_DB.Raw("SELECT AVG(COALESCE(silent_days,0)) FROM ai_wx_customer").Scan(&avgSilent)
	if avgSilent != nil {
		health.AvgSilentDays = *avgSilent
	}

	var deals int64
	global.GVA_DB.Table("ai_wx_customer").Where("stage = ?", "已成交").Count(&deals)
	health.DealCount = int(deals)

	var valid int64
	global.GVA_DB.Table("ai_wx_customer").Where("stage <> ? AND stage <> ?", "无效", "已成交").Count(&valid)
	health.ValidCount = int(valid)

	return map[string]any{"sources": sources, "health": health, "sides": sideStats}
}

// ImportResult 导入结果
type ImportResult struct {
	Inserted int      `json:"inserted"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors"`
}

// ImportLeads 从 Excel 导入线索
// 支持表头：昵称/姓名, 手机/电话/手机号, 微信/微信号, 城市/地区, 备注/内容, 来源, 关键词
func (leadService *LeadService) ImportLeads(reader io.Reader) (ImportResult, error) {
	result := ImportResult{Errors: make([]string, 0)}

	file, err := excelize.OpenReader(reader)
	if err != nil {
		return result, fmt.Errorf("打不开 Excel: %s", err.Error())
	}
	defer file.Close()

	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return result, fmt.Errorf("Excel 里没有工作表")
	}

	rows, err := file.GetRows(sheets[0])
	if err != nil || len(rows) < 2 {
		return result, fmt.Errorf("内容为空或只有表头")
	}

	header := rows[0]
	col := map[string]int{}

	for i, h := range header {
		key := strings.TrimSpace(strings.ToLower(h))

		switch {
		case strings.Contains(key, "昵称"), strings.Contains(key, "姓名"), strings.Contains(key, "客户"):
			col["name"] = i
		case strings.Contains(key, "手机"), strings.Contains(key, "电话"):
			col["phone"] = i
		case strings.Contains(key, "微信"):
			col["wechat"] = i
		case strings.Contains(key, "城市"), strings.Contains(key, "地区"):
			col["city"] = i
		case strings.Contains(key, "备注"), strings.Contains(key, "内容"):
			col["remark"] = i
		case strings.Contains(key, "来源"), strings.Contains(key, "平台"):
			col["source"] = i
		case strings.Contains(key, "关键词"), strings.Contains(key, "行业"):
			col["keyword"] = i
		}
	}

	if _, ok := col["name"]; !ok {
		return result, fmt.Errorf("表头里找不到「昵称/姓名」列")
	}

	get := func(row []string, field string) string {
		idx, ok := col[field]
		if !ok || idx >= len(row) {
			return ""
		}

		return strings.TrimSpace(row[idx])
	}

	for idx, row := range rows[1:] {
		name := get(row, "name")

		if name == "" {
			result.Skipped++
			continue
		}

		// 同名已存在就跳过
		var exist int64
		global.GVA_DB.Table("ai_lead").Where("contact_name = ?", name).Count(&exist)

		if exist > 0 {
			result.Skipped++
			continue
		}

		source := get(row, "source")
		if source == "" {
			source = "import"
		}

		empty := ""
		zero := int64(0)
		nowText := time.Now().Format("2006-01-02 15:04:05")

		item := leadModel.Lead{
			ContactName: &name,
			Source:      &source,
			Company:     &name,
			Industry:    &empty,
			Avatar:      &empty,
			VideoTitle:  &empty,
			ExtractInfo: &empty,
			Score:       &zero,
			Status:      &zero,
			DmScript:    "",
			CreatedAt:   &nowText,
			UpdatedAt:   &nowText,
		}

		if phone := get(row, "phone"); phone != "" {
			item.Phone = &phone
		}

		if wechat := get(row, "wechat"); wechat != "" {
			item.Wechat = &wechat
		}

		if city := get(row, "city"); city != "" {
			item.City = &city
		}

		if remark := get(row, "remark"); remark != "" {
			item.Remark = &remark
		}

		if keyword := get(row, "keyword"); keyword != "" {
			item.Keyword = &keyword
		}

		grade, reason, ready := GradeLead(&item)
		item.Grade = grade
		item.GradeReason = reason

		if ready {
			item.ContactReady = 1
		}

		if err := global.GVA_DB.Table("ai_lead").Create(&item).Error; err != nil {
			result.Failed++

			if len(result.Errors) < 10 {
				result.Errors = append(result.Errors, fmt.Sprintf("第 %d 行: %s", idx+2, err.Error()))
			}

			continue
		}

		result.Inserted++
	}

	return result, nil
}

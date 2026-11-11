package lead

import (
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// WorkbenchItem 待办项
// Judgement 今日判断（一句话结论 + 依据 + 建议）
type Judgement struct {
	Headline    string   `json:"headline"`
	Basis       []string `json:"basis"`
	Suggestion  string   `json:"suggestion"`
	Impact      string   `json:"impact"`
}

type WorkbenchItem struct {
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Count    int    `json:"count"`
	Action   string `json:"action"`
}

// WorkbenchLead 线索简表
type WorkbenchLead struct {
	Id        uint   `json:"Id"`
	Company   string `json:"Company"`
	Keyword   string `json:"Keyword"`
	City      string `json:"City"`
	Score     int    `json:"Score"`
	Grade     string `json:"Grade"`
	Ready     int    `json:"Ready"`
	Source    string `json:"Source"`
	CreatedAt string `json:"CreatedAt"`
}

// WorkbenchCustomer 客户简表
type WorkbenchCustomer struct {
	Id         uint   `json:"Id"`
	NickName   string `json:"NickName"`
	Tier       string `json:"Tier"`
	Stage      string `json:"Stage"`
	Score      int    `json:"Score"`
	SilentDays int    `json:"SilentDays"`
	NextAction string `json:"NextAction"`
}

// Workbench 今日工作台
type Workbench struct {
	Today          string              `json:"today"`
	HandoffCount   int                 `json:"handoffCount"`
	HotCount       int                 `json:"hotCount"`
	SilentCount    int                 `json:"silentCount"`
	LeadCount      int                 `json:"leadCount"`
	NewLeadToday   int                 `json:"newLeadToday"`
	ReplyToday     int                 `json:"replyToday"`
	Items          []WorkbenchItem     `json:"items"`
	Handoffs       []leadModel.WxHandoff `json:"handoffs"`
	HotCustomers   []WorkbenchCustomer `json:"hotCustomers"`
	SilentCustomers []WorkbenchCustomer `json:"silentCustomers"`
	NewLeads       []WorkbenchLead     `json:"newLeads"`
	Judgement      Judgement           `json:"judgement"`
}

// GetWorkbench 汇总今天该干的活
func (leadService *LeadService) GetWorkbench() (Workbench, error) {
	day := time.Now().Format("2006-01-02")

	result := Workbench{
		Today:           day,
		Items:           make([]WorkbenchItem, 0, 4),
		Handoffs:        make([]leadModel.WxHandoff, 0, 5),
		HotCustomers:    make([]WorkbenchCustomer, 0, 8),
		SilentCustomers: make([]WorkbenchCustomer, 0, 8),
		NewLeads:        make([]WorkbenchLead, 0, 10),
	}

	// 待接管工单
	var handoffs []leadModel.WxHandoff
	global.GVA_DB.Where("status = 0").Order("urgency desc, id desc").Limit(5).Find(&handoffs)
	result.Handoffs = handoffs

	var handoffTotal int64
	global.GVA_DB.Model(&leadModel.WxHandoff{}).Where("status = 0").Count(&handoffTotal)
	result.HandoffCount = int(handoffTotal)

	// 客户：按分层 + 沉默天数
	var customers []leadModel.WxCustomer
	global.GVA_DB.Order("score desc, id desc").Find(&customers)

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

	for _, row := range seen {
		item := WorkbenchCustomer{
			Id:         uint(row.Id),
			NickName:   row.NickName,
			Tier:       row.Tier,
			Stage:      row.Stage,
			Score:      row.Score,
			SilentDays: row.SilentDays,
			NextAction: row.NextAction,
		}

		if row.SilentDays >= 7 {
			result.SilentCustomers = append(result.SilentCustomers, item)
		} else if row.Tier == "A" || row.Tier == "B" {
			result.HotCustomers = append(result.HotCustomers, item)
		}
	}

	result.HotCount = len(result.HotCustomers)
	result.SilentCount = len(result.SilentCustomers)

	// 新线索（未处理）
	var readyCount64 int64
	global.GVA_DB.Model(&leadModel.Lead{}).Where("status = 0 AND grade IN ('S','A') AND contact_ready = 1").Count(&readyCount64)
	readyCount := int(readyCount64)

	var leadTotal int64
	global.GVA_DB.Model(&leadModel.Lead{}).Where("status = 0 AND grade IN ('S','A')").Count(&leadTotal)
	result.LeadCount = int(leadTotal)

	var todayLeads int64
	global.GVA_DB.Model(&leadModel.Lead{}).Where("DATE(created_at) = ?", day).Count(&todayLeads)
	result.NewLeadToday = int(todayLeads)

	var leads []leadModel.Lead
	global.GVA_DB.Where("status = 0 AND grade IN ('S','A')").Order("contact_ready desc, grade asc, score desc, id desc").Limit(10).Find(&leads)

	for _, row := range leads {
		createdAt := ""
		if row.CreatedAt != nil {
			value := *row.CreatedAt
			if len(value) >= 16 {
				createdAt = strings.ReplaceAll(value[5:16], "T", " ")
			} else {
				createdAt = value
			}
		}

		id := uint(0)
		if row.Id != nil {
			id = uint(*row.Id)
		}

		score := 0
		if row.Score != nil {
			score = int(*row.Score)
		}

		result.NewLeads = append(result.NewLeads, WorkbenchLead{
			Id:        id,
			Company:   derefString(row.ContactName),
			Keyword:   derefString(row.Keyword),
			City:      derefString(row.City),
			Score:     score,
			Grade:     row.Grade,
			Ready:     row.ContactReady,
			Source:    derefString(row.Source),
			CreatedAt: createdAt,
		})
	}

	// 今日 AI 回复数
	var replyToday int64
	global.GVA_DB.Model(&leadModel.WxFollowup{}).
		Where("kind = ? AND DATE(created_at) = ?", "自动回复", day).Count(&replyToday)
	result.ReplyToday = int(replyToday)

	result.Judgement = leadService.buildJudgement(result)

	result.Items = []WorkbenchItem{
		{Kind: "handoff", Title: "待接管工单", Subtitle: "对方需要人工处理", Count: result.HandoffCount, Action: "去处理"},
		{Kind: "hot", Title: "高价值客户", Subtitle: "A/B 类，正在推进", Count: result.HotCount, Action: "去跟进"},
		{Kind: "silent", Title: "沉默客户", Subtitle: "超过 7 天没说话", Count: result.SilentCount, Action: "去唤醒"},
		{Kind: "lead", Title: "优先线索", Subtitle: fmt.Sprintf("A1 可直接联系 %d · A2 先互动 %d", readyCount, result.LeadCount-readyCount), Count: result.LeadCount, Action: "去筛选"},
	}

	return result, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
// buildJudgement 按规则生成今日判断（先算事实，再给建议）
func (leadService *LeadService) buildJudgement(wb Workbench) Judgement {
	basis := make([]string, 0, 4)
	headline := ""
	suggestion := ""
	impact := ""

	switch {
	case wb.HandoffCount > 0:
		headline = fmt.Sprintf("有 %d 个客户在等人工回复，先处理掉他们", wb.HandoffCount)
		basis = append(basis, fmt.Sprintf("待接管工单 %d 个", wb.HandoffCount))
		suggestion = "打开待接管，逐条回复后再让 AI 接手"
		impact = "拖得越久，客户越可能去找别家"
	case wb.LeadCount > 0 && wb.HotCount == 0:
		headline = fmt.Sprintf("有 %d 条优先线索还没转成客户，今天该推进", wb.LeadCount)
		basis = append(basis, fmt.Sprintf("优先线索 %d 条", wb.LeadCount))
		basis = append(basis, "客户池里没有 A/B 类在推进")
		suggestion = "挑 3-5 条 A 类线索，转客户并发出第一条私信"
		impact = "线索放着不动，7 天后基本就凉了"
	case wb.SilentCount > 0:
		headline = fmt.Sprintf("有 %d 个客户超过 7 天没说话，需要唤醒", wb.SilentCount)
		basis = append(basis, fmt.Sprintf("沉默客户 %d 个", wb.SilentCount))
		suggestion = "用报价格式发一条跟进，带具体规格和运费"
		impact = "沉默超过 14 天，回复率会掉一半"
	case wb.HotCount > 0:
		headline = fmt.Sprintf("有 %d 个高价值客户在推进，今天把报价推进到下一步", wb.HotCount)
		basis = append(basis, fmt.Sprintf("高价值客户 %d 个", wb.HotCount))
		suggestion = "优先给 A 类客户明确报价与结算方式"
		impact = "报价拖一天，成交周期平均延长 2-3 天"
	default:
		headline = "今天没有紧急事项，建议补充新线索"
		basis = append(basis, "没有待接管、沉默客户和优先线索")
		suggestion = "采集一轮小红书 / 抖音，把线索池补起来"
		impact = "线索是源头，断了后面全是空的"
	}

	if wb.ReplyToday > 0 {
		basis = append(basis, fmt.Sprintf("AI 今日已自动回复 %d 条", wb.ReplyToday))
	}

	return Judgement{Headline: headline, Basis: basis, Suggestion: suggestion, Impact: impact}
}
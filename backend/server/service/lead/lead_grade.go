package lead

import (
	"fmt"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// GradeLead 按规则给线索分级（S/A/B/C/D）
//
// 规则优先级：
//  1. 有手机号或微信号 → 直接加分（能直接联系到人）
//  2. 评分高（账号获客通常 75+，评论获客通常 30 以下）
//  3. 关键词命中买方意图词（求购/批发/价格/多少钱/怎么卖…）
//  4. 命中卖方/同行词（招聘/加盟/代理/供应…）→ 降级
func GradeLead(item *leadModel.Lead) (string, string, bool) {
	score := 0
	reasons := make([]string, 0, 4)

	base := 0
	if item.Score != nil {
		base = int(*item.Score)
	}

	// 基础分（0-60）
	switch {
	case base >= 75:
		score += 60
		reasons = append(reasons, fmt.Sprintf("评分%d", base))
	case base >= 50:
		score += 45
		reasons = append(reasons, fmt.Sprintf("评分%d", base))
	case base >= 30:
		score += 28
		reasons = append(reasons, fmt.Sprintf("评分%d", base))
	default:
		score += 10
		reasons = append(reasons, fmt.Sprintf("评分%d", base))
	}

	// 可直接联系（+25）
	hasPhone := item.Phone != nil && len(strings.TrimSpace(*item.Phone)) >= 7
	hasWechat := item.Wechat != nil && strings.TrimSpace(*item.Wechat) != ""

	if hasPhone || hasWechat {
		score += 25
		reasons = append(reasons, "有联系方式")
	}

	// 意图匹配：按「对方角色」分别看
	//
	// 业务是两个方向都做的：
	//   buyer    对方是买我们货的客户（餐饮、超市、二批）→ 看买方意图
	//   supplier 对方是供货给我们的（养殖场、批发商）→ 看供货意图
	// side 为空时两边都试，取加成更高的那个，并在依据里写明方向。
	text := strings.ToLower(leadText(item))

	buyWords := []string{"求购", "批发", "多少钱", "怎么卖", "价格", "报价", "拿货", "进货", "采购", "要货"}
	sellWords := []string{"货源", "供货", "养殖场", "日产", "出货", "批发价", "一件代发", "厂家", "供应", "库存"}
	noiseWords := []string{"招聘", "加盟", "代理招商", "收购", "招商"}

	// 明确的"求价/要货"动作，用来区分"客户在问"和"同行在晒"
	askWords := []string{"求购", "多少钱", "怎么卖", "啥价", "什么价", "报价", "拿货", "进货", "采购", "要货", "怎么批", "能不能发"}

	hitBuy := containsAny(text, buyWords)
	hitSell := containsAny(text, sellWords)

	side := ""
	if item.Side != nil {
		side = strings.TrimSpace(*item.Side)
	}

	if side == "" {
		// 没标方向：按文本猜（买方词或供货词都算有效），并把猜到的方向写回
		switch {
		case hitBuy && !hitSell:
			side = "buyer"
		case hitSell && !hitBuy:
			side = "supplier"
		case hitBuy && hitSell:
			side = "buyer" // 两边都像时，默认当客户
		}

		if side != "" {
			item.Side = &side
		}
	}

	switch side {
	case "supplier":
		if hitSell {
			score += 15
			reasons = append(reasons, "供货意图")
		}

		if hitBuy {
			score -= 10
			reasons = append(reasons, "像在找货（同行居多）")
		}

	default: // buyer 或未定
		if hitBuy {
			score += 15
			reasons = append(reasons, "买方意图")
		}

		// 对方是客户时，出现"供应/货源/厂家"这类词说明他更像同行卖家；
		// 但如果同时在问价（多少钱/怎么卖/要货），那是客户在问我们能不能供，不能罚。
		if hitSell && !containsAny(text, askWords) {
			score -= 20
			reasons = append(reasons, "疑似同行")
		}
	}

	for _, word := range noiseWords {
		if strings.Contains(text, word) {
			score -= 20
			reasons = append(reasons, "疑似同行")
			break
		}
	}

	// 有地区信息（+5）
	if item.City != nil && strings.TrimSpace(*item.City) != "" {
		score += 5
	}

	grade := "D"

	switch {
	case score >= 85:
		grade = "S"
	case score >= 65:
		grade = "A"
	case score >= 45:
		grade = "B"
	case score >= 25:
		grade = "C"
	}

	// 有手机号或微信号 = 能直接联系，A1/S1；否则只能先互动，A2/S2
	contactReady := hasPhone || hasWechat

	return grade, strings.Join(reasons, "，"), contactReady
}


func leadText(item *leadModel.Lead) string {
	parts := make([]string, 0, 4)

	for _, value := range []*string{item.ContactName, item.Keyword, item.Remark, item.VideoTitle, item.ExtractInfo} {
		if value != nil && *value != "" {
			parts = append(parts, *value)
		}
	}

	return strings.Join(parts, " ")
}

// GradeAllLeads 批量给线索分级（返回更新条数）
func (leadService *LeadService) GradeAllLeads() (int, error) {
	var leads []leadModel.Lead

	if err := global.GVA_DB.Find(&leads).Error; err != nil {
		return 0, err
	}

	updated := 0

	for index := range leads {
		// 先记住原来的方向：GradeLead 会就地改 Side，改完再读就比不出来了
		oldSide := ""
		if leads[index].Side != nil {
			oldSide = strings.TrimSpace(*leads[index].Side)
		}

		grade, reason, contactReady := GradeLead(&leads[index])

		ready := 0
		if contactReady {
			ready = 1
		}

		// 分级时推断出来的方向（谁买谁卖）
		newSide := ""
		if leads[index].Side != nil {
			newSide = strings.TrimSpace(*leads[index].Side)
		}

		// 方向没变、等级没变就跳过；注意之前只比等级，
		// 导致已有线索的 side 永远不会被补上
		if leads[index].Grade == grade && leads[index].GradeReason == reason &&
			leads[index].ContactReady == ready && oldSide == newSide {
			continue
		}

		values := map[string]any{
			"grade":         grade,
			"grade_reason":  reason,
			"contact_ready": ready,
			"side":          newSide,
		}

		global.GVA_DB.Model(&leadModel.Lead{}).Where("id = ?", leads[index].Id).Updates(values)

		updated++
	}

	return updated, nil
}

// ApplyLeadContact 抖音私信里对方给了号码时，回填到线索并重新分级
func (leadService *LeadService) ApplyLeadContact(nick, contact string) (string, error) {
	if nick == "" || contact == "" {
		return "", nil
	}

	var lead leadModel.Lead

	err := global.GVA_DB.Where("contact_name = ?", nick).Order("id desc").First(&lead).Error
	if err != nil {
		return "", err
	}

	updates := map[string]any{
		"extract_info": contact,
	}

	if len(contact) >= 7 && contact[0] == '1' {
		updates["phone"] = contact
	} else {
		updates["wechat"] = contact
	}

	if err := global.GVA_DB.Model(&leadModel.Lead{}).Where("id = ?", lead.Id).Updates(updates).Error; err != nil {
		return "", err
	}

	// 重新读一遍再分级，保证 phone/wechat 已生效
	global.GVA_DB.First(&lead, lead.Id)

	grade, reason, ready := GradeLead(&lead)

	readyFlag := 0
	if ready {
		readyFlag = 1
	}

	global.GVA_DB.Model(&leadModel.Lead{}).Where("id = ?", lead.Id).Updates(map[string]any{
		"grade":         grade,
		"grade_reason":  reason,
		"contact_ready": readyFlag,
	})

	return grade, nil
}
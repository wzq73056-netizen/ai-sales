package lead

import (
	"regexp"
	"strings"

	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 手机号：11 位，允许带空格/短横线
var phonePattern = regexp.MustCompile(`1[3-9]\d[\s-]?\d{4}[\s-]?\d{4}`)

// 微信号：必须带前缀，避免把普通英文单词当微信号
var wechatPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:微信号?|weixin|wechat|vx|v信|威信|薇信)\s*[:：]?\s*([A-Za-z][-_A-Za-z0-9]{4,19})`),
	regexp.MustCompile(`(?i)(?:加我|联系我)\s*[:：]?\s*([A-Za-z][-_A-Za-z0-9]{4,19})`),
}

// extractContacts 从线索文本里抽手机号 / 微信号
//
// 采集只拿到评论原文和昵称，联系方式常常就写在里面（“加我 vx abc123”）。
// 之前这些线索全部计为「无联系方式」，A 类永远出不来。
func extractContacts(text string) (phone, wechat string) {
	if strings.TrimSpace(text) == "" {
		return "", ""
	}

	if m := phonePattern.FindString(text); m != "" {
		phone = strings.NewReplacer(" ", "", "-", "").Replace(m)
	}

	for _, pattern := range wechatPatterns {
		if m := pattern.FindStringSubmatch(text); len(m) > 1 {
			wechat = m[1]

			break
		}
	}

	return phone, wechat
}

// ExtractLeadContacts 批量从线索文本里回填联系方式
//
// 返回：扫描条数、回填手机号数、回填微信号数
func (leadService *LeadService) ExtractLeadContacts() (scanned, phones, wechats int, err error) {
	var leads []leadModel.Lead

	// 只扫还没有联系方式的线索
	if err = global.GVA_DB.
		Where("(phone IS NULL OR phone = '') AND (wechat IS NULL OR wechat = '')").
		Find(&leads).Error; err != nil {
		return 0, 0, 0, err
	}

	for i := range leads {
		item := &leads[i]
		scanned++

		text := strings.Join([]string{
			deref(item.Remark),
			deref(item.ExtractInfo),
			deref(item.ContactName),
			deref(item.Company),
		}, " ")

		phone, wechat := extractContacts(text)
		if phone == "" && wechat == "" {
			continue
		}

		updates := map[string]any{}

		if phone != "" {
			updates["phone"] = phone
			phones++
		}

		if wechat != "" {
			updates["wechat"] = wechat
			wechats++
		}

		if len(updates) > 0 {
			if err = global.GVA_DB.Model(&leadModel.Lead{}).Where("id = ?", item.Id).Updates(updates).Error; err != nil {
				return scanned, phones, wechats, err
			}
		}
	}

	return scanned, phones, wechats, nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

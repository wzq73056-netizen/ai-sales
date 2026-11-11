package lead

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

// GetSystemAlerts 系统告警（微信掉登录等）
// @Tags Lead
// @Summary 系统告警
// @Router /lead/getSystemAlerts [get]
func (leadApi *LeadApi) GetSystemAlerts(c *gin.Context) {
	alerts := leadService.GetSystemAlerts()

	response.OkWithDetailed(gin.H{
		"list":  alerts,
		"count": len(alerts),
		"wx":    leadService.GetWxLoginStatus(),
	}, "获取成功", c)
}
// ExtractLeadContacts 从线索文本里回填手机号/微信号
// @Tags Lead
// @Summary 抽取线索联系方式
// @Router /lead/extractLeadContacts [post]
func (leadApi *LeadApi) ExtractLeadContacts(c *gin.Context) {
	scanned, phones, wechats, err := leadService.ExtractLeadContacts()
	if err != nil {
		response.FailWithMessage(err.Error(), c)

		return
	}

	// 回填完立刻重新分级，等级后缀会从 A2 变成 A1
	_, _ = leadService.GradeAllLeads()

	response.OkWithDetailed(gin.H{
		"scanned": scanned,
		"phones":  phones,
		"wechats": wechats,
	}, "抽取完成", c)
}
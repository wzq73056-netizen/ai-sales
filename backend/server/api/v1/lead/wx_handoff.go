package lead

import (
	"path/filepath"
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
	"github.com/gin-gonic/gin"
)

// CheckWxIntent 判断消息是否需要转人工
// @Tags Lead
// @Summary 微信消息转人工判定
// @Security ApiKeyAuth
// @Router /lead/checkWxIntent [post]
func (leadApi *LeadApi) CheckWxIntent(c *gin.Context) {
	var req struct {
		Text    string `json:"Text"`
		History string `json:"History"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	result, err := leadService.CheckIntent(c.Request.Context(), req.Text, req.History)
	if err != nil {
		response.FailWithMessage("判定失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "判定完成", c)
}

// CreateWxHandoff 建转人工工单
// @Tags Lead
// @Summary 创建微信转人工工单
// @Security ApiKeyAuth
// @Router /lead/createWxHandoff [post]
func (leadApi *LeadApi) CreateWxHandoff(c *gin.Context) {
	var req leadModel.WxHandoff

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	item, err := leadService.CreateHandoff(req)
	if err != nil {
		response.FailWithMessage("创建失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(item, "已建单", c)
}

// GetWxHandoffList 待接管列表
// @Tags Lead
// @Summary 微信转人工列表
// @Security ApiKeyAuth
// @Router /lead/getWxHandoffList [get]
func (leadApi *LeadApi) GetWxHandoffList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status, _ := strconv.Atoi(c.DefaultQuery("status", "0"))

	list, total, err := leadService.GetWxHandoffList(page, pageSize, status)
	if err != nil {
		response.FailWithMessage("获取失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

// ClaimWxHandoff 认领工单
// @Tags Lead
// @Summary 认领微信转人工工单
// @Security ApiKeyAuth
// @Router /lead/claimWxHandoff [post]
func (leadApi *LeadApi) ClaimWxHandoff(c *gin.Context) {
	var req struct {
		Id   int64  `json:"Id"`
		User string `json:"User"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.ClaimWxHandoff(req.Id, req.User); err != nil {
		response.FailWithMessage("认领失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("已认领", c)
}

// ResolveWxHandoff 处理完成
// @Tags Lead
// @Summary 处理微信转人工工单
// @Security ApiKeyAuth
// @Router /lead/resolveWxHandoff [post]
func (leadApi *LeadApi) ResolveWxHandoff(c *gin.Context) {
	var req struct {
		Id     int64 `json:"Id"`
		Status int   `json:"Status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.ResolveWxHandoff(req.Id, req.Status); err != nil {
		response.FailWithMessage("处理失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("已处理", c)
}

// GetWxHandoffCount 待接管数量（角标用）
// @Tags Lead
// @Summary 待接管数量
// @Security ApiKeyAuth
// @Router /lead/getWxHandoffCount [get]
func (leadApi *LeadApi) GetWxHandoffCount(c *gin.Context) {
	list, total, err := leadService.GetWxHandoffList(1, 5, 0)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{
		"Pending": total,
		"Latest":  list,
	}, "获取成功", c)
}
// UpdateWxHandoffResult 更新转微信跟进状态
// @Tags Lead
// @Summary 更新转微信跟进
// @Security ApiKeyAuth
// @Router /lead/updateWxHandoffResult [post]
func (leadApi *LeadApi) UpdateWxHandoffResult(c *gin.Context) {
	var req struct {
		Id          int64  `json:"Id"`
		WechatAdded int    `json:"WechatAdded"`
		Result      string `json:"Result"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.UpdateWxHandoffResult(req.Id, req.WechatAdded, req.Result); err != nil {
		response.FailWithMessage("更新失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("已更新", c)
}
// GetAiPausedChats 被暂停 AI 的会话列表
// @Tags Lead
// @Summary AI 暂停的会话
// @Security ApiKeyAuth
// @Router /lead/getAiPausedChats [get]
func (leadApi *LeadApi) GetAiPausedChats(c *gin.Context) {
	response.OkWithDetailed(gin.H{"chats": leadService.GetAiPausedChats()}, "获取成功", c)
}

// ResumeAi 手动恢复 AI
// @Tags Lead
// @Summary 恢复AI自动回复
// @Security ApiKeyAuth
// @Router /lead/resumeAi [post]
func (leadApi *LeadApi) ResumeAi(c *gin.Context) {
	var req struct {
		Id int64 `json:"Id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.ResumeAi(req.Id); err != nil {
		response.FailWithMessage("恢复失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("AI 已恢复该会话", c)
}
// GetFunnelStats 转化漏斗统计
// @Tags Lead
// @Summary 转化漏斗
// @Security ApiKeyAuth
// @Router /lead/getFunnelStats [get]
func (leadApi *LeadApi) GetFunnelStats(c *gin.Context) {
	stats, err := leadService.GetFunnelStats()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(stats, "获取成功", c)
}
// GetWorkbench 今日工作台
// @Tags Lead
// @Summary 今日工作台
// @Security ApiKeyAuth
// @Router /lead/getWorkbench [get]
func (leadApi *LeadApi) GetWorkbench(c *gin.Context) {
	data, err := leadService.GetWorkbench()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(data, "获取成功", c)
}
// GradeAllLeads 批量给线索分级
// @Tags Lead
// @Summary 线索自动分级
// @Security ApiKeyAuth
// @Router /lead/gradeAllLeads [post]
func (leadApi *LeadApi) GradeAllLeads(c *gin.Context) {
	updated, err := leadService.GradeAllLeads()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"updated": updated}, "分级完成", c)
}
// ApplyLeadContact 抖音私信拿到号码后回填线索
// @Tags Lead
// @Summary 回填线索联系方式
// @Security ApiKeyAuth
// @Router /lead/applyLeadContact [post]
func (leadApi *LeadApi) ApplyLeadContact(c *gin.Context) {
	var req struct {
		NickName string `json:"NickName"`
		Contact  string `json:"Contact"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	grade, err := leadService.ApplyLeadContact(req.NickName, req.Contact)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"grade": grade}, "已回填", c)
}
// GenLeadDmScript 生成单条线索的私信话术
// @Tags Lead
// @Summary 生成私信话术
// @Security ApiKeyAuth
// @Router /lead/genLeadDmScript [post]
func (leadApi *LeadApi) GenLeadDmScript(c *gin.Context) {
	var req struct {
		Id uint `json:"Id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	script, err := leadService.GenLeadDmScript(c.Request.Context(), req.Id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"script": script}, "已生成", c)
}

// GenLeadDmScripts 批量生成话术
// @Tags Lead
// @Summary 批量生成私信话术
// @Security ApiKeyAuth
// @Router /lead/genLeadDmScripts [post]
func (leadApi *LeadApi) GenLeadDmScripts(c *gin.Context) {
	var req struct {
		Grade  string `json:"Grade"`
		Limit  int    `json:"Limit"`
		Source string `json:"Source"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	done, err := leadService.GenLeadDmScripts(c.Request.Context(), req.Grade, req.Limit, req.Source)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"generated": done}, "已生成", c)
}
// StartXhsCollect 触发小红书采集
// @Tags Lead
// @Summary 小红书采集
// @Security ApiKeyAuth
// @Router /lead/startXhsCollect [post]
func (leadApi *LeadApi) StartXhsCollect(c *gin.Context) {
	var req struct {
		Keywords string `json:"Keywords"`
		Notes    int    `json:"Notes"`
		Limit    int    `json:"Limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	status, err := leadService.StartXhsCollect(req.Keywords, req.Notes, req.Limit)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(status, "已开始采集", c)
}

// GetXhsCollectStatus 小红书采集状态
// @Tags Lead
// @Summary 小红书采集状态
// @Security ApiKeyAuth
// @Router /lead/getXhsCollectStatus [get]
func (leadApi *LeadApi) GetXhsCollectStatus(c *gin.Context) {
	response.OkWithDetailed(leadService.GetXhsCollectStatus(), "获取成功", c)
}

// GetXhsLeadStats 小红书线索统计
// @Tags Lead
// @Summary 小红书线索统计
// @Security ApiKeyAuth
// @Router /lead/getXhsLeadStats [get]
func (leadApi *LeadApi) GetXhsLeadStats(c *gin.Context) {
	response.OkWithDetailed(leadService.XhsLeadStats(), "获取成功", c)
}
// GetXhsRuns 小红书采集历史
// @Tags Lead
// @Summary 小红书采集历史
// @Security ApiKeyAuth
// @Router /lead/getXhsRuns [get]
func (leadApi *LeadApi) GetXhsRuns(c *gin.Context) {
	response.OkWithDetailed(gin.H{"list": leadService.GetXhsRuns(20)}, "获取成功", c)
}
// SendXhsDm 小红书主动私信
// @Tags Lead
// @Summary 小红书主动私信
// @Security ApiKeyAuth
// @Router /lead/sendXhsDm [post]
func (leadApi *LeadApi) SendXhsDm(c *gin.Context) {
	var req struct {
		NickName string `json:"NickName"`
		Message  string `json:"Message"`
		AiHint   string `json:"AiHint"`
		Send     bool   `json:"Send"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	result, err := leadService.SendXhsDm(req.NickName, req.Message, req.AiHint, req.Send)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "已执行", c)
}
// GetXhsSchedule 小红书定时采集配置
// @Tags Lead
// @Summary 小红书定时配置
// @Security ApiKeyAuth
// @Router /lead/getXhsSchedule [get]
func (leadApi *LeadApi) GetXhsSchedule(c *gin.Context) {
	response.OkWithDetailed(leadService.GetXhsSchedule(), "获取成功", c)
}

// SaveXhsSchedule 保存小红书定时配置
// @Tags Lead
// @Summary 保存小红书定时配置
// @Security ApiKeyAuth
// @Router /lead/saveXhsSchedule [post]
func (leadApi *LeadApi) SaveXhsSchedule(c *gin.Context) {
	var req leadModel.XhsSchedule

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.SaveXhsSchedule(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已保存", c)
}
// SendDyDm 抖音主动私信
// @Tags Lead
// @Summary 抖音主动私信
// @Security ApiKeyAuth
// @Router /lead/sendDyDm [post]
func (leadApi *LeadApi) SendDyDm(c *gin.Context) {
	var req struct {
		NickName string `json:"NickName"`
		Message  string `json:"Message"`
		AiHint   string `json:"AiHint"`
		Send     bool   `json:"Send"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	result, err := leadService.SendDyDm(req.NickName, req.Message, req.AiHint, req.Send)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "已执行", c)
}
// SendXhsProactiveDm 给线索主动私信（默认预览）
// @Tags Lead
// @Summary 小红书线索主动私信
// @Security ApiKeyAuth
// @Router /lead/sendXhsProactiveDm [post]
func (leadApi *LeadApi) SendXhsProactiveDm(c *gin.Context) {
	var req struct {
		LeadId  uint   `json:"LeadId"`
		Message string `json:"Message"`
		AiHint  string `json:"AiHint"`
		Send    bool   `json:"Send"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	result, err := leadService.SendXhsProactiveDm(req.LeadId, req.Message, req.AiHint, req.Send)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "已执行", c)
}
// GetInbox 统一收件箱
// @Tags Lead
// @Summary 统一收件箱
// @Security ApiKeyAuth
// @Router /lead/getInbox [get]
func (leadApi *LeadApi) GetInbox(c *gin.Context) {
	limit := 60

	if value := c.Query("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			limit = parsed
		}
	}

	result, err := leadService.GetInbox(c.Query("channel"), limit)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "获取成功", c)
}
// RecordDmMessages 监听器上报会话消息
// @Tags Lead
// @Summary 记录私信消息
// @Security ApiKeyAuth
// @Router /lead/recordDmMessages [post]
func (leadApi *LeadApi) RecordDmMessages(c *gin.Context) {
	var req struct {
		Platform string                         `json:"Platform"`
		DeviceId int                            `json:"DeviceId"`
		Items    []leadModel.DmMessageInput   `json:"Items"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	n, err := leadService.RecordDmMessages(req.Platform, req.DeviceId, req.Items)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"inserted": n}, "已记录", c)
}

// GetDmConversations 会话聚合
// @Tags Lead
// @Summary 私信会话列表
// @Security ApiKeyAuth
// @Router /lead/getDmConversations [get]
func (leadApi *LeadApi) GetDmConversations(c *gin.Context) {
	response.OkWithDetailed(gin.H{"list": leadService.GetDmConversations(c.Query("platform"), 60)}, "获取成功", c)
}

// GetScrapeKeywords 关键词库
// @Tags Lead
// @Summary 采集关键词
// @Security ApiKeyAuth
// @Router /lead/getScrapeKeywords [get]
func (leadApi *LeadApi) GetScrapeKeywords(c *gin.Context) {
	response.OkWithDetailed(gin.H{"list": leadService.GetScrapeKeywords(c.Query("platform"))}, "获取成功", c)
}

// SaveScrapeKeyword 新增关键词
// @Tags Lead
// @Summary 新增采集关键词
// @Security ApiKeyAuth
// @Router /lead/saveScrapeKeyword [post]
func (leadApi *LeadApi) SaveScrapeKeyword(c *gin.Context) {
	var req leadModel.ScrapeKeyword
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.SaveScrapeKeyword(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已保存", c)
}

// DeleteScrapeKeyword 删除关键词
// @Tags Lead
// @Summary 删除采集关键词
// @Security ApiKeyAuth
// @Router /lead/deleteScrapeKeyword [post]
func (leadApi *LeadApi) DeleteScrapeKeyword(c *gin.Context) {
	var req struct {
		Id uint `json:"Id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.DeleteScrapeKeyword(req.Id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已删除", c)
}

// GetScrapeBlacklist 黑名单
// @Tags Lead
// @Summary 采集黑名单
// @Security ApiKeyAuth
// @Router /lead/getScrapeBlacklist [get]
func (leadApi *LeadApi) GetScrapeBlacklist(c *gin.Context) {
	response.OkWithDetailed(gin.H{"list": leadService.GetScrapeBlacklist()}, "获取成功", c)
}

// SaveScrapeBlacklist 新增黑名单
// @Tags Lead
// @Summary 新增黑名单
// @Security ApiKeyAuth
// @Router /lead/saveScrapeBlacklist [post]
func (leadApi *LeadApi) SaveScrapeBlacklist(c *gin.Context) {
	var req leadModel.ScrapeBlacklist
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.SaveScrapeBlacklist(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已保存", c)
}

// DeleteScrapeBlacklist 删除黑名单
// @Tags Lead
// @Summary 删除黑名单
// @Security ApiKeyAuth
// @Router /lead/deleteScrapeBlacklist [post]
func (leadApi *LeadApi) DeleteScrapeBlacklist(c *gin.Context) {
	var req struct {
		Id uint `json:"Id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.DeleteScrapeBlacklist(req.Id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已删除", c)
}

// ExportLeads 导出线索 Excel
// @Tags Lead
// @Summary 导出线索
// @Security ApiKeyAuth
// @Router /lead/exportLeads [get]
func (leadApi *LeadApi) ExportLeads(c *gin.Context) {
	path, err := leadService.ExportLeads(c.Query("source"), c.Query("grade"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	c.FileAttachment(path, filepath.Base(path))
}
// ConvertLead 线索转客户
// @Tags Lead
// @Summary 线索转客户
// @Security ApiKeyAuth
// @Router /lead/convertLead [post]
func (leadApi *LeadApi) ConvertLead(c *gin.Context) {
	var req struct {
		Id uint `json:"Id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	result, err := leadService.ConvertLeadToCustomer(req.Id)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "已转客户", c)
}
// GetScriptStats 话术效果统计
// @Tags Lead
// @Summary 话术回复率
// @Security ApiKeyAuth
// @Router /lead/getScriptStats [get]
func (leadApi *LeadApi) GetScriptStats(c *gin.Context) {
	response.OkWithDetailed(leadService.GetScriptStats(), "获取成功", c)
}
// AddWxFriend 微信加好友
// @Tags Lead
// @Summary 微信加好友
// @Security ApiKeyAuth
// @Router /lead/addWxFriend [post]
func (leadApi *LeadApi) AddWxFriend(c *gin.Context) {
	var req struct {
		Id     uint   `json:"Id"`
		Remark string `json:"Remark"`
		Send   bool   `json:"Send"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	result, err := leadService.AddWxFriend(req.Id, req.Remark, req.Send)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "已执行", c)
}
// GetWxAddConfig 加好友配置
// @Tags Lead
// @Summary 加好友配置
// @Security ApiKeyAuth
// @Router /lead/getWxAddConfig [get]
func (leadApi *LeadApi) GetWxAddConfig(c *gin.Context) {
	response.OkWithDetailed(gin.H{
		"config": leadService.GetWxAddConfig(),
		"stats":  leadService.GetWxAddStats(),
	}, "获取成功", c)
}

// SaveWxAddConfig 保存加好友配置
// @Tags Lead
// @Summary 保存加好友配置
// @Security ApiKeyAuth
// @Router /lead/saveWxAddConfig [post]
func (leadApi *LeadApi) SaveWxAddConfig(c *gin.Context) {
	var req leadModel.WxAddConfig

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.SaveWxAddConfig(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已保存", c)
}

// EnqueueWxAdd 加入加好友队列
// @Tags Lead
// @Summary 加入加好友队列
// @Security ApiKeyAuth
// @Router /lead/enqueueWxAdd [post]
func (leadApi *LeadApi) EnqueueWxAdd(c *gin.Context) {
	var req struct {
		LeadIds []uint `json:"LeadIds"`
		Remark  string `json:"Remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	result, err := leadService.EnqueueWxAdd(req.LeadIds, req.Remark)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "已入队", c)
}

// GetWxAddQueue 队列
// @Tags Lead
// @Summary 加好友队列
// @Security ApiKeyAuth
// @Router /lead/getWxAddQueue [get]
func (leadApi *LeadApi) GetWxAddQueue(c *gin.Context) {
	response.OkWithDetailed(gin.H{
		"list":  leadService.GetWxAddQueue(60),
		"stats": leadService.GetWxAddStats(),
	}, "获取成功", c)
}

// RunWxAddOnce 手动执行一次
// @Tags Lead
// @Summary 手动执行一次加好友
// @Security ApiKeyAuth
// @Router /lead/runWxAddOnce [post]
func (leadApi *LeadApi) RunWxAddOnce(c *gin.Context) {
	result, err := leadService.RunWxAddOnce()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "已执行", c)
}
// ClaimWxAddJob 监听器领取加好友任务
// @Tags Lead
// @Summary 领取加好友任务
// @Security ApiKeyAuth
// @Router /lead/claimWxAddJob [get]
func (leadApi *LeadApi) ClaimWxAddJob(c *gin.Context) {
	job, reason := leadService.ClaimWxAddJob()

	if job == nil {
		response.OkWithDetailed(gin.H{"job": nil, "reason": reason}, "无任务", c)
		return
	}

	response.OkWithDetailed(gin.H{"job": job}, "已领取", c)
}

// FinishWxAddJob 监听器回报加好友结果
// @Tags Lead
// @Summary 回报加好友结果
// @Security ApiKeyAuth
// @Router /lead/finishWxAddJob [post]
func (leadApi *LeadApi) FinishWxAddJob(c *gin.Context) {
	var req struct {
		Id     uint   `json:"Id"`
		State  string `json:"State"`
		Detail string `json:"Detail"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.FinishWxAddJob(req.Id, req.State, req.Detail); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已记录", c)
}
// GetFollowupReminders 跟进提醒
// @Tags Lead
// @Summary 跟进提醒看板
// @Security ApiKeyAuth
// @Router /lead/getFollowupReminders [get]
func (leadApi *LeadApi) GetFollowupReminders(c *gin.Context) {
	response.OkWithDetailed(leadService.GetFollowupReminders(), "获取成功", c)
}
// ConvertLeads 批量线索转客户
// @Tags Lead
// @Summary 批量转客户
// @Security ApiKeyAuth
// @Router /lead/convertLeads [post]
func (leadApi *LeadApi) ConvertLeads(c *gin.Context) {
	var req struct {
		Ids []uint `json:"Ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(leadService.ConvertLeads(req.Ids), "已处理", c)
}

// GetLeadAnalytics 线索来源分布 + 跟进健康度
// @Tags Lead
// @Summary 报表分析
// @Security ApiKeyAuth
// @Router /lead/getLeadAnalytics [get]
func (leadApi *LeadApi) GetLeadAnalytics(c *gin.Context) {
	response.OkWithDetailed(leadService.GetLeadAnalytics(), "获取成功", c)
}

// ImportLeads 导入线索 Excel
// @Tags Lead
// @Summary 导入线索
// @Security ApiKeyAuth
// @Router /lead/importLeads [post]
func (leadApi *LeadApi) ImportLeads(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithMessage("没有收到文件", c)
		return
	}

	f, err := file.Open()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	defer f.Close()

	result, err := leadService.ImportLeads(f)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "导入完成", c)
}
package lead

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	leadSvc "github.com/flipped-aurora/gin-vue-admin/server/service/lead"
	"github.com/gin-gonic/gin"
)

// SyncWxCustomer 同步微信客户价值
// @Tags Lead
// @Summary 同步微信客户跟进价值
// @Security ApiKeyAuth
// @Router /lead/syncWxCustomer [post]
func (leadApi *LeadApi) SyncWxCustomer(c *gin.Context) {
	var req struct {
		Items []leadSvc.WxChatSnapshot `json:"Items"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	count, err := leadService.SyncWxCustomers(c.Request.Context(), req.Items)
	if err != nil {
		response.FailWithMessage("同步失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"synced": count}, "同步完成", c)
}

// GetWxCustomerList 客户跟进列表
// @Tags Lead
// @Summary 微信客户跟进列表
// @Security ApiKeyAuth
// @Router /lead/getWxCustomerList [get]
func (leadApi *LeadApi) GetWxCustomerList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	minScore, _ := strconv.Atoi(c.DefaultQuery("minScore", "0"))

	list, total, err := leadService.GetWxCustomerList(page, pageSize, minScore, c.Query("tier"), c.Query("stage"), c.Query("keyword"), c.Query("side"), c.Query("sort"), c.Query("order"))
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

// MarkWxCustomerFollow 标记跟进状态
// @Tags Lead
// @Summary 标记微信客户跟进状态
// @Security ApiKeyAuth
// @Router /lead/markWxCustomerFollow [post]
func (leadApi *LeadApi) MarkWxCustomerFollow(c *gin.Context) {
	var req struct {
		Id     int64 `json:"Id"`
		Status int   `json:"Status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.MarkWxCustomerFollow(req.Id, req.Status); err != nil {
		response.FailWithMessage("更新失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("已更新", c)
}

// GetWxCustomerGrouped 客户列表（按昵称合并）
// @Tags Lead
// @Summary 客户列表（合并）
// @Security ApiKeyAuth
// @Router /lead/getWxCustomerGrouped [get]
func (leadApi *LeadApi) GetWxCustomerGrouped(c *gin.Context) {
	minScore, _ := strconv.Atoi(c.DefaultQuery("minScore", "0"))
	list, err := leadService.GetWxCustomerGrouped(minScore, 100)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"list": list, "total": len(list)}, "获取成功", c)
}

// GetWxCustomerDetail 客户详情 + 跟进记录
// @Tags Lead
// @Summary 客户详情
// @Security ApiKeyAuth
// @Router /lead/getWxCustomerDetail [get]
func (leadApi *LeadApi) GetWxCustomerDetail(c *gin.Context) {
	nick := c.Query("nick")
	if nick == "" {
		response.FailWithMessage("缺少 nick", c)
		return
	}

	detail, err := leadService.GetWxCustomerDetail(nick)
	if err != nil {
		response.FailWithMessage("未找到该客户", c)
		return
	}

	response.OkWithDetailed(detail, "获取成功", c)
}

// AddWxFollowup 新增跟进记录
// @Tags Lead
// @Summary 新增跟进
// @Security ApiKeyAuth
// @Router /lead/addWxFollowup [post]
func (leadApi *LeadApi) AddWxFollowup(c *gin.Context) {
	var req struct {
		NickName string `json:"NickName"`
		ChatId   string `json:"ChatId"`
		Kind     string `json:"Kind"`
		Content  string `json:"Content"`
		MediaUrl string `json:"MediaUrl"`
		Operator string `json:"Operator"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if req.Content == "" {
		response.FailWithMessage("内容不能为空", c)
		return
	}

	if err := leadService.AddWxFollowup(req.NickName, req.ChatId, req.Kind, req.Content, req.MediaUrl, req.Operator); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已记录", c)
}
// UpdateWxCustomer 更新客户信息
// @Tags Lead
// @Summary 更新客户阶段/跟进计划
// @Security ApiKeyAuth
// @Router /lead/updateWxCustomer [post]
func (leadApi *LeadApi) UpdateWxCustomer(c *gin.Context) {
	var req struct {
		Id           uint   `json:"Id"`
		Stage        string `json:"Stage"`
		NextFollowAt string `json:"NextFollowAt"`
		NextAction   string `json:"NextAction"`
		Tags         string `json:"Tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.UpdateWxCustomer(req.Id, req.Stage, req.NextFollowAt, req.NextAction, req.Tags); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已保存", c)
}
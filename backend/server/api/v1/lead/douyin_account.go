package lead

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
	leadSvc "github.com/flipped-aurora/gin-vue-admin/server/service/lead"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SaveDouyinAccounts 批量保存账号线索（采集脚本回调用）
// @Tags Lead
// @Summary 批量保存抖音账号线索
// @Security ApiKeyAuth
// @Router /lead/saveDouyinAccounts [post]
func (leadApi *LeadApi) SaveDouyinAccounts(c *gin.Context) {
	var req struct {
		Accounts []leadModel.DouyinAccount `json:"Accounts"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	created, err := leadService.SaveDouyinAccounts(req.Accounts)
	if err != nil {
		global.GVA_LOG.Error("保存账号线索失败!", zap.Error(err))
		response.FailWithMessage("保存失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"created": created, "total": len(req.Accounts)}, "保存成功", c)
}

// GetDouyinAccountList 账号线索列表
// @Tags Lead
// @Summary 抖音账号线索列表
// @Security ApiKeyAuth
// @Router /lead/getDouyinAccountList [get]
func (leadApi *LeadApi) GetDouyinAccountList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	onlyCompetitor, _ := strconv.Atoi(c.DefaultQuery("competitor", "0"))
	sortBy := c.DefaultQuery("sortBy", "fans")

	list, total, err := leadService.GetDouyinAccountList(page, pageSize, keyword, onlyCompetitor, sortBy)
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

// UpdateDouyinAccountStatus 更新账号线索状态
// @Tags Lead
// @Summary 更新抖音账号线索状态
// @Security ApiKeyAuth
// @Router /lead/updateDouyinAccountStatus [put]
func (leadApi *LeadApi) UpdateDouyinAccountStatus(c *gin.Context) {
	var req struct {
		Id     int64  `json:"Id"`
		Status int    `json:"Status"`
		Remark string `json:"Remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.UpdateDouyinAccount(req.Id, req.Status, req.Remark); err != nil {
		response.FailWithMessage("更新失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("已更新", c)
}

// DeleteDouyinAccount 删除账号线索
// @Tags Lead
// @Summary 删除抖音账号线索
// @Security ApiKeyAuth
// @Router /lead/deleteDouyinAccount [delete]
func (leadApi *LeadApi) DeleteDouyinAccount(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Query("Id"), 10, 64)

	if err := leadService.DeleteDouyinAccount(id); err != nil {
		response.FailWithMessage("删除失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

// ClassifyComments 评论意向判定（大模型）
// @Tags Lead
// @Summary 评论线索判定
// @Security ApiKeyAuth
// @Router /lead/classifyComments [post]
func (leadApi *LeadApi) ClassifyComments(c *gin.Context) {
	var req struct {
		Items []leadSvc.ClassifyItem `json:"Items"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	results, err := leadService.ClassifyComments(req.Items)
	if err != nil {
		global.GVA_LOG.Error("评论判定失败!", zap.Error(err))
		response.FailWithMessage("判定失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(results, "判定成功", c)
}

// SyncAccountsToLeads 账号线索同步进线索池
// @Tags Lead
// @Summary 账号线索同步到线索池
// @Security ApiKeyAuth
// @Router /lead/syncAccountsToLeads [post]
func (leadApi *LeadApi) SyncAccountsToLeads(c *gin.Context) {
	created, err := leadService.SyncAccountsToLeads()
	if err != nil {
		response.FailWithMessage("同步失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"created": created}, "同步完成", c)
}
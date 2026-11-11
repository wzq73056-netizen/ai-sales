package lead

import (
	
	"github.com/flipped-aurora/gin-vue-admin/server/global"
    "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
    "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
    leadReq "github.com/flipped-aurora/gin-vue-admin/server/model/lead/request"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type LeadApi struct {}



// CreateLead 创建获客线索
// @Tags Lead
// @Summary 创建获客线索
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data body lead.Lead true "创建获客线索"
// @Success 200 {object} response.Response{msg=string} "创建成功"
// @Router /lead/createLead [post]
func (leadApi *LeadApi) CreateLead(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	var lead lead.Lead
	err := c.ShouldBindJSON(&lead)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = leadService.CreateLead(ctx,&lead)
	if err != nil {
        global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败:" + err.Error(), c)
		return
	}
    response.OkWithMessage("创建成功", c)
}

// DeleteLead 删除获客线索
// @Tags Lead
// @Summary 删除获客线索
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data body lead.Lead true "删除获客线索"
// @Success 200 {object} response.Response{msg=string} "删除成功"
// @Router /lead/deleteLead [delete]
func (leadApi *LeadApi) DeleteLead(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	Id := c.Query("Id")
	err := leadService.DeleteLead(ctx,Id)
	if err != nil {
        global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败:" + err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// DeleteLeadByIds 批量删除获客线索
// @Tags Lead
// @Summary 批量删除获客线索
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{msg=string} "批量删除成功"
// @Router /lead/deleteLeadByIds [delete]
func (leadApi *LeadApi) DeleteLeadByIds(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	Ids := c.QueryArray("Ids[]")
	err := leadService.DeleteLeadByIds(ctx,Ids)
	if err != nil {
        global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败:" + err.Error(), c)
		return
	}
	response.OkWithMessage("批量删除成功", c)
}

// UpdateLead 更新获客线索
// @Tags Lead
// @Summary 更新获客线索
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data body lead.Lead true "更新获客线索"
// @Success 200 {object} response.Response{msg=string} "更新成功"
// @Router /lead/updateLead [put]
func (leadApi *LeadApi) UpdateLead(c *gin.Context) {
    // 从ctx获取标准context进行业务行为
    ctx := c.Request.Context()

	var lead lead.Lead
	err := c.ShouldBindJSON(&lead)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = leadService.UpdateLead(ctx,lead)
	if err != nil {
        global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败:" + err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// FindLead 用id查询获客线索
// @Tags Lead
// @Summary 用id查询获客线索
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param Id query int true "用id查询获客线索"
// @Success 200 {object} response.Response{data=lead.Lead,msg=string} "查询成功"
// @Router /lead/findLead [get]
func (leadApi *LeadApi) FindLead(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	Id := c.Query("Id")
	relead, err := leadService.GetLead(ctx,Id)
	if err != nil {
        global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败:" + err.Error(), c)
		return
	}
	response.OkWithData(relead, c)
}
// GetLeadList 分页获取获客线索列表
// @Tags Lead
// @Summary 分页获取获客线索列表
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data query leadReq.LeadSearch true "分页获取获客线索列表"
// @Success 200 {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router /lead/getLeadList [get]
func (leadApi *LeadApi) GetLeadList(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	var pageInfo leadReq.LeadSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := leadService.GetLeadInfoList(ctx,pageInfo)
	if err != nil {
	    global.GVA_LOG.Error("获取失败!", zap.Error(err))
        response.FailWithMessage("获取失败:" + err.Error(), c)
        return
    }
    response.OkWithDetailed(response.PageResult{
        List:     list,
        Total:    total,
        Page:     pageInfo.Page,
        PageSize: pageInfo.PageSize,
    }, "获取成功", c)
}

// GetLeadPublic 不需要鉴权的获客线索接口
// @Tags Lead
// @Summary 不需要鉴权的获客线索接口
// @Accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{data=object,msg=string} "获取成功"
// @Router /lead/getLeadPublic [get]
func (leadApi *LeadApi) GetLeadPublic(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

    // 此接口不需要鉴权
    // 示例为返回了一个固定的消息接口，一般本接口用于C端服务，需要自己实现业务逻辑
    leadService.GetLeadPublic(ctx)
    response.OkWithDetailed(gin.H{
       "info": "不需要鉴权的获客线索接口信息",
    }, "获取成功", c)
}

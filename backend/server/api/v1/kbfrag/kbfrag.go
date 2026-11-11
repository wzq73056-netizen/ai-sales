package kbfrag

import (
	
	"github.com/flipped-aurora/gin-vue-admin/server/global"
    "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
    "github.com/flipped-aurora/gin-vue-admin/server/model/kbfrag"
    kbfragReq "github.com/flipped-aurora/gin-vue-admin/server/model/kbfrag/request"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type KnowledgeFragmentApi struct {}



// CreateKnowledgeFragment 创建知识库分片
// @Tags KnowledgeFragment
// @Summary 创建知识库分片
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data body kbfrag.KnowledgeFragment true "创建知识库分片"
// @Success 200 {object} response.Response{msg=string} "创建成功"
// @Router /kbfrag/createKnowledgeFragment [post]
func (kbfragApi *KnowledgeFragmentApi) CreateKnowledgeFragment(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	var kbfrag kbfrag.KnowledgeFragment
	err := c.ShouldBindJSON(&kbfrag)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = kbfragService.CreateKnowledgeFragment(ctx,&kbfrag)
	if err != nil {
        global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败:" + err.Error(), c)
		return
	}
    response.OkWithMessage("创建成功", c)
}

// DeleteKnowledgeFragment 删除知识库分片
// @Tags KnowledgeFragment
// @Summary 删除知识库分片
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data body kbfrag.KnowledgeFragment true "删除知识库分片"
// @Success 200 {object} response.Response{msg=string} "删除成功"
// @Router /kbfrag/deleteKnowledgeFragment [delete]
func (kbfragApi *KnowledgeFragmentApi) DeleteKnowledgeFragment(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	Id := c.Query("Id")
	err := kbfragService.DeleteKnowledgeFragment(ctx,Id)
	if err != nil {
        global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败:" + err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// DeleteKnowledgeFragmentByIds 批量删除知识库分片
// @Tags KnowledgeFragment
// @Summary 批量删除知识库分片
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{msg=string} "批量删除成功"
// @Router /kbfrag/deleteKnowledgeFragmentByIds [delete]
func (kbfragApi *KnowledgeFragmentApi) DeleteKnowledgeFragmentByIds(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	Ids := c.QueryArray("Ids[]")
	err := kbfragService.DeleteKnowledgeFragmentByIds(ctx,Ids)
	if err != nil {
        global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败:" + err.Error(), c)
		return
	}
	response.OkWithMessage("批量删除成功", c)
}

// UpdateKnowledgeFragment 更新知识库分片
// @Tags KnowledgeFragment
// @Summary 更新知识库分片
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data body kbfrag.KnowledgeFragment true "更新知识库分片"
// @Success 200 {object} response.Response{msg=string} "更新成功"
// @Router /kbfrag/updateKnowledgeFragment [put]
func (kbfragApi *KnowledgeFragmentApi) UpdateKnowledgeFragment(c *gin.Context) {
    // 从ctx获取标准context进行业务行为
    ctx := c.Request.Context()

	var kbfrag kbfrag.KnowledgeFragment
	err := c.ShouldBindJSON(&kbfrag)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = kbfragService.UpdateKnowledgeFragment(ctx,kbfrag)
	if err != nil {
        global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败:" + err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// FindKnowledgeFragment 用id查询知识库分片
// @Tags KnowledgeFragment
// @Summary 用id查询知识库分片
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param Id query int true "用id查询知识库分片"
// @Success 200 {object} response.Response{data=kbfrag.KnowledgeFragment,msg=string} "查询成功"
// @Router /kbfrag/findKnowledgeFragment [get]
func (kbfragApi *KnowledgeFragmentApi) FindKnowledgeFragment(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	Id := c.Query("Id")
	rekbfrag, err := kbfragService.GetKnowledgeFragment(ctx,Id)
	if err != nil {
        global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败:" + err.Error(), c)
		return
	}
	response.OkWithData(rekbfrag, c)
}
// GetKnowledgeFragmentList 分页获取知识库分片列表
// @Tags KnowledgeFragment
// @Summary 分页获取知识库分片列表
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data query kbfragReq.KnowledgeFragmentSearch true "分页获取知识库分片列表"
// @Success 200 {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router /kbfrag/getKnowledgeFragmentList [get]
func (kbfragApi *KnowledgeFragmentApi) GetKnowledgeFragmentList(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	var pageInfo kbfragReq.KnowledgeFragmentSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := kbfragService.GetKnowledgeFragmentInfoList(ctx,pageInfo)
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

// GetKnowledgeFragmentPublic 不需要鉴权的知识库分片接口
// @Tags KnowledgeFragment
// @Summary 不需要鉴权的知识库分片接口
// @Accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{data=object,msg=string} "获取成功"
// @Router /kbfrag/getKnowledgeFragmentPublic [get]
func (kbfragApi *KnowledgeFragmentApi) GetKnowledgeFragmentPublic(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

    // 此接口不需要鉴权
    // 示例为返回了一个固定的消息接口，一般本接口用于C端服务，需要自己实现业务逻辑
    kbfragService.GetKnowledgeFragmentPublic(ctx)
    response.OkWithDetailed(gin.H{
       "info": "不需要鉴权的知识库分片接口信息",
    }, "获取成功", c)
}

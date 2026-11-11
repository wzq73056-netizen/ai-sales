package kb

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
    "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
    "github.com/flipped-aurora/gin-vue-admin/server/model/kb"
    kbReq "github.com/flipped-aurora/gin-vue-admin/server/model/kb/request"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type KnowledgeApi struct {}



// CreateKnowledge 创建KnowledgeBase
// @Tags Knowledge
// @Summary 创建KnowledgeBase
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data body kb.Knowledge true "创建KnowledgeBase"
// @Success 200 {object} response.Response{msg=string} "创建成功"
// @Router /kb/createKnowledge [post]
func (kbApi *KnowledgeApi) CreateKnowledge(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	var kb kb.Knowledge
	err := c.ShouldBindJSON(&kb)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = kbService.CreateKnowledge(ctx,&kb)
	if err != nil {
        global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败:" + err.Error(), c)
		return
	}
    response.OkWithMessage("创建成功", c)
}

// DeleteKnowledge 删除KnowledgeBase
// @Tags Knowledge
// @Summary 删除KnowledgeBase
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data body kb.Knowledge true "删除KnowledgeBase"
// @Success 200 {object} response.Response{msg=string} "删除成功"
// @Router /kb/deleteKnowledge [delete]
func (kbApi *KnowledgeApi) DeleteKnowledge(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	Id := c.Query("Id")
	err := kbService.DeleteKnowledge(ctx,Id)
	if err != nil {
        global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败:" + err.Error(), c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// DeleteKnowledgeByIds 批量删除KnowledgeBase
// @Tags Knowledge
// @Summary 批量删除KnowledgeBase
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{msg=string} "批量删除成功"
// @Router /kb/deleteKnowledgeByIds [delete]
func (kbApi *KnowledgeApi) DeleteKnowledgeByIds(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	// 兼容三种数组序列化：Ids[]=1&Ids[]=2 / Ids=1&Ids=2 / Ids[0]=1&Ids[1]=2
	Ids := c.QueryArray("Ids[]")
	if len(Ids) == 0 {
		Ids = c.QueryArray("Ids")
	}
	if len(Ids) == 0 {
		for i := 0; ; i++ {
			v := c.Query("Ids[" + strconv.Itoa(i) + "]")
			if v == "" {
				break
			}
			Ids = append(Ids, v)
		}
	}
	if len(Ids) == 0 {
		response.FailWithMessage("未选择要删除的记录", c)
		return
	}
	err := kbService.DeleteKnowledgeByIds(ctx,Ids)
	if err != nil {
        global.GVA_LOG.Error("批量删除失败!", zap.Error(err))
		response.FailWithMessage("批量删除失败:" + err.Error(), c)
		return
	}
	response.OkWithMessage("批量删除成功", c)
}

// UpdateKnowledge 更新KnowledgeBase
// @Tags Knowledge
// @Summary 更新KnowledgeBase
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data body kb.Knowledge true "更新KnowledgeBase"
// @Success 200 {object} response.Response{msg=string} "更新成功"
// @Router /kb/updateKnowledge [put]
func (kbApi *KnowledgeApi) UpdateKnowledge(c *gin.Context) {
    // 从ctx获取标准context进行业务行为
    ctx := c.Request.Context()

	var kb kb.Knowledge
	err := c.ShouldBindJSON(&kb)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = kbService.UpdateKnowledge(ctx,kb)
	if err != nil {
        global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败:" + err.Error(), c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// FindKnowledge 用id查询KnowledgeBase
// @Tags Knowledge
// @Summary 用id查询KnowledgeBase
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param Id query int true "用id查询KnowledgeBase"
// @Success 200 {object} response.Response{data=kb.Knowledge,msg=string} "查询成功"
// @Router /kb/findKnowledge [get]
func (kbApi *KnowledgeApi) FindKnowledge(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	Id := c.Query("Id")
	rekb, err := kbService.GetKnowledge(ctx,Id)
	if err != nil {
        global.GVA_LOG.Error("查询失败!", zap.Error(err))
		response.FailWithMessage("查询失败:" + err.Error(), c)
		return
	}
	response.OkWithData(rekb, c)
}
// GetKnowledgeList 分页获取KnowledgeBase列表
// @Tags Knowledge
// @Summary 分页获取KnowledgeBase列表
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param data query kbReq.KnowledgeSearch true "分页获取KnowledgeBase列表"
// @Success 200 {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router /kb/getKnowledgeList [get]
func (kbApi *KnowledgeApi) GetKnowledgeList(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

	var pageInfo kbReq.KnowledgeSearch
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	list, total, err := kbService.GetKnowledgeInfoList(ctx,pageInfo)
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

// GetKnowledgePublic 不需要鉴权的KnowledgeBase接口
// @Tags Knowledge
// @Summary 不需要鉴权的KnowledgeBase接口
// @Accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{data=object,msg=string} "获取成功"
// @Router /kb/getKnowledgePublic [get]
func (kbApi *KnowledgeApi) GetKnowledgePublic(c *gin.Context) {
    // 创建业务用Context
    ctx := c.Request.Context()

    // 此接口不需要鉴权
    // 示例为返回了一个固定的消息接口，一般本接口用于C端服务，需要自己实现业务逻辑
    kbService.GetKnowledgePublic(ctx)
    response.OkWithDetailed(gin.H{
       "info": "不需要鉴权的KnowledgeBase接口信息",
    }, "获取成功", c)
}

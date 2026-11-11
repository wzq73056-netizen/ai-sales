package lead

import (
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
	leadSvc "github.com/flipped-aurora/gin-vue-admin/server/service/lead"
	"github.com/gin-gonic/gin"
)

// CheckDouyinDm 发送前查重
// @Tags Lead
// @Summary 抖音私信发送前查重
// @Security ApiKeyAuth
// @Router /lead/checkDouyinDm [post]
func (leadApi *LeadApi) CheckDouyinDm(c *gin.Context) {
	var req struct {
		ProfileUrl string `json:"ProfileUrl"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	result, err := leadService.CheckDouyinDm(req.ProfileUrl)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "查询成功", c)
}

// SaveDouyinDm 记录私信发送结果
// @Tags Lead
// @Summary 记录抖音私信发送
// @Security ApiKeyAuth
// @Router /lead/saveDouyinDm [post]
func (leadApi *LeadApi) SaveDouyinDm(c *gin.Context) {
	var req leadModel.DouyinDm

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	item, err := leadService.SaveDouyinDm(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(item, "已记录", c)
}

// GetDouyinDmList 私信记录列表
// @Tags Lead
// @Summary 抖音私信记录
// @Security ApiKeyAuth
// @Router /lead/getDouyinDmList [get]
func (leadApi *LeadApi) GetDouyinDmList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := leadService.GetDouyinDmList(page, pageSize)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

// UpdateDmReply 记录对方回复
// @Tags Lead
// @Summary 记录私信回复
// @Security ApiKeyAuth
// @Router /lead/updateDmReply [post]
func (leadApi *LeadApi) UpdateDmReply(c *gin.Context) {
	var req struct {
		Id        int64  `json:"Id"`
		ReplyText string `json:"ReplyText"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.UpdateDmReply(req.Id, req.ReplyText); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已记录回复", c)
}

// GetPendingDmList 待检查回复的私信
// @Tags Lead
// @Summary 待检查回复的私信列表
// @Security ApiKeyAuth
// @Router /lead/getPendingDmList [get]
func (leadApi *LeadApi) GetPendingDmList(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	list, err := leadService.GetPendingDmList(limit)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(list, "获取成功", c)
}
// GenDmReply AI 生成私信回复
// @Tags Lead
// @Summary AI 生成抖音私信回复
// @Security ApiKeyAuth
// @Router /lead/genDmReply [post]
func (leadApi *LeadApi) GenDmReply(c *gin.Context) {
	var req leadSvc.DmReplyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	reply, err := leadService.GenDmReply(c.Request.Context(), req)
	if err != nil {
		response.FailWithMessage("生成失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"Reply": reply}, "生成成功", c)
}
// GetDmControl 读取私信自动化开关
// @Tags Lead
// @Summary 私信自动化开关
// @Security ApiKeyAuth
// @Router /lead/getDmControl [get]
func (leadApi *LeadApi) GetDmControl(c *gin.Context) {
	response.OkWithDetailed(leadService.GetDmStatus(), "获取成功", c)
}

// SetDmControl 设置私信自动化开关
// @Tags Lead
// @Summary 设置私信自动化开关
// @Security ApiKeyAuth
// @Router /lead/setDmControl [post]
func (leadApi *LeadApi) SetDmControl(c *gin.Context) {
	var req leadSvc.DmControl

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.SetDmControl(req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 留一点时间让守护进程写出首轮状态
	time.Sleep(500 * time.Millisecond)

	response.OkWithDetailed(leadService.GetDmStatus(), "已保存", c)
}
package lead

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
	"github.com/gin-gonic/gin"
)

// GetDouyinDevices 抖音账号列表
// @Tags Lead
// @Summary 抖音账号列表
// @Security ApiKeyAuth
// @Router /lead/getDouyinDevices [get]
func (leadApi *LeadApi) GetDouyinDevices(c *gin.Context) {
	list, err := leadService.GetDouyinDevices(c.Query("platform"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"list": list}, "获取成功", c)
}

// SaveDouyinDevice 保存抖音账号
// @Tags Lead
// @Summary 保存抖音账号
// @Security ApiKeyAuth
// @Router /lead/saveDouyinDevice [post]
func (leadApi *LeadApi) SaveDouyinDevice(c *gin.Context) {
	var item leadModel.DouyinDevice

	if err := c.ShouldBindJSON(&item); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if item.CdpPort == 0 {
		response.FailWithMessage("CDP 端口不能为空", c)
		return
	}

	saved, err := leadService.SaveDouyinDevice(item)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(saved, "保存成功", c)
}

// DeleteDouyinDevice 删除抖音账号
// @Tags Lead
// @Summary 删除抖音账号
// @Security ApiKeyAuth
// @Router /lead/deleteDouyinDevice [post]
func (leadApi *LeadApi) DeleteDouyinDevice(c *gin.Context) {
	var req struct {
		Id uint `json:"Id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.DeleteDouyinDevice(req.Id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已删除", c)
}

// GetDouyinWhitelist 白名单列表
// @Tags Lead
// @Summary 抖音私信白名单
// @Security ApiKeyAuth
// @Router /lead/getDouyinWhitelist [get]
func (leadApi *LeadApi) GetDouyinWhitelist(c *gin.Context) {
	deviceId := 0

	if value := c.Query("deviceId"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			deviceId = parsed
		}
	}

	list, err := leadService.GetDouyinWhitelist(uint(deviceId), c.Query("platform"))
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(gin.H{"list": list}, "获取成功", c)
}

// SaveDouyinWhitelist 保存白名单
// @Tags Lead
// @Summary 保存抖音私信白名单
// @Security ApiKeyAuth
// @Router /lead/saveDouyinWhitelist [post]
func (leadApi *LeadApi) SaveDouyinWhitelist(c *gin.Context) {
	var item leadModel.DouyinWhitelist

	if err := c.ShouldBindJSON(&item); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if item.DouyinId == "" || item.ProfileUrl == "" {
		response.FailWithMessage("抖音号和主页地址不能为空", c)
		return
	}

	item.Enabled = 1

	saved, err := leadService.SaveDouyinWhitelist(item)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(saved, "保存成功", c)
}

// DeleteDouyinWhitelist 删除白名单
// @Tags Lead
// @Summary 删除抖音私信白名单
// @Security ApiKeyAuth
// @Router /lead/deleteDouyinWhitelist [post]
func (leadApi *LeadApi) DeleteDouyinWhitelist(c *gin.Context) {
	var req struct {
		Id uint `json:"Id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.DeleteDouyinWhitelist(req.Id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已删除", c)
}

// TouchDouyinDevice 监听器心跳
// @Tags Lead
// @Summary 抖音账号心跳
// @Security ApiKeyAuth
// @Router /lead/touchDouyinDevice [post]
func (leadApi *LeadApi) TouchDouyinDevice(c *gin.Context) {
	var req struct {
		Id uint `json:"Id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.TouchDouyinDevice(req.Id); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("ok", c)
}

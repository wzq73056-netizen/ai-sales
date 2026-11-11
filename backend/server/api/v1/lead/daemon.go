package lead

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

// GetDaemons 监听进程状态
// @Tags Lead
// @Summary 获取抖音/小红书/微信监听进程状态
// @Router /lead/getDaemons [get]
func (leadApi *LeadApi) GetDaemons(c *gin.Context) {
	list, err := leadService.GetDaemons()
	if err != nil {
		response.FailWithMessage(err.Error(), c)

		return
	}

	response.OkWithDetailed(gin.H{"list": list}, "获取成功", c)
}

// StartDaemon 启动监听进程
// @Tags Lead
// @Summary 启动监听进程
// @Router /lead/startDaemon [post]
func (leadApi *LeadApi) StartDaemon(c *gin.Context) {
	var req struct {
		Key string `json:"key"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
		response.FailWithMessage("缺少进程标识", c)

		return
	}

	msg, err := leadService.StartDaemon(req.Key)
	if err != nil {
		response.FailWithMessage(err.Error(), c)

		return
	}

	list, _ := leadService.GetDaemons()

	response.OkWithDetailed(gin.H{"list": list, "msg": msg}, msg, c)
}

// StopDaemon 停止监听进程
// @Tags Lead
// @Summary 停止监听进程
// @Router /lead/stopDaemon [post]
func (leadApi *LeadApi) StopDaemon(c *gin.Context) {
	var req struct {
		Key string `json:"key"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" {
		response.FailWithMessage("缺少进程标识", c)

		return
	}

	msg, err := leadService.StopDaemon(req.Key)
	if err != nil {
		response.FailWithMessage(err.Error(), c)

		return
	}

	list, _ := leadService.GetDaemons()

	response.OkWithDetailed(gin.H{"list": list, "msg": msg}, msg, c)
}

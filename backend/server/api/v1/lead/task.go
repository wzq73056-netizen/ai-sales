package lead

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CreateDouyinTask 新建抖音采集任务
// @Tags Lead
// @Summary 新建抖音采集任务
// @Security ApiKeyAuth
// @Router /lead/createDouyinTask [post]
func (leadApi *LeadApi) CreateDouyinTask(c *gin.Context) {
	var req struct {
		Industry         string `json:"Industry"`
		Name             string `json:"Name"`
		DeviceId         uint   `json:"DeviceId"`
		CrawlType        int    `json:"CrawlType"`
		Keywords         string `json:"Keywords"`
		VideosPerKeyword int    `json:"VideosPerKeyword"`
		MaxDays          int    `json:"MaxDays"`
		LimitN           int    `json:"LimitN"`
		City             string `json:"City"`
		ExcludeWords     string `json:"ExcludeWords"`
		ScheduleType     string `json:"ScheduleType"`
		Schedule         string `json:"Schedule"`
		AutoStart        bool   `json:"AutoStart"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	task := leadModel.DouyinTask{
		Industry:         req.Industry,
		Name:             req.Name,
		DeviceId:         req.DeviceId,
		CrawlType:        req.CrawlType,
		Keywords:         req.Keywords,
		VideosPerKeyword: req.VideosPerKeyword,
		MaxDays:          req.MaxDays,
		LimitN:           req.LimitN,
		City:             req.City,
		ExcludeWords:     req.ExcludeWords,
		ScheduleType:     req.ScheduleType,
		Schedule:         req.Schedule,
	}

	created, err := leadService.CreateDouyinTask(&task, req.AutoStart)
	if err != nil {
		global.GVA_LOG.Error("创建任务失败!", zap.Error(err))
		response.FailWithMessage("创建失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(created, "创建成功", c)
}

// GetDouyinTaskList 任务列表
// @Tags Lead
// @Summary 抖音采集任务列表
// @Security ApiKeyAuth
// @Router /lead/getDouyinTaskList [get]
func (leadApi *LeadApi) GetDouyinTaskList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	list, total, err := leadService.GetDouyinTaskList(page, pageSize)
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

// StartDouyinTask 启动任务
// @Tags Lead
// @Summary 启动抖音采集任务
// @Security ApiKeyAuth
// @Router /lead/startDouyinTask [post]
func (leadApi *LeadApi) StartDouyinTask(c *gin.Context) {
	var req struct {
		Id int64 `json:"Id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := leadService.StartDouyinTask(req.Id); err != nil {
		response.FailWithMessage("启动失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("已启动", c)
}

// StopDouyinTask 停止任务
// @Tags Lead
// @Summary 停止抖音采集任务
// @Security ApiKeyAuth
// @Router /lead/stopDouyinTask [post]
func (leadApi *LeadApi) StopDouyinTask(c *gin.Context) {
	if err := leadService.StopCurrent(); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("已停止", c)
}

// DeleteDouyinTask 删除任务
// @Tags Lead
// @Summary 删除抖音采集任务
// @Security ApiKeyAuth
// @Router /lead/deleteDouyinTask [delete]
func (leadApi *LeadApi) DeleteDouyinTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Query("Id"), 10, 64)

	if err := leadService.DeleteDouyinTask(id); err != nil {
		response.FailWithMessage("删除失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

// GetRunningLog 当前运行日志（实时）
// @Tags Lead
// @Summary 当前运行任务日志
// @Security ApiKeyAuth
// @Router /lead/runningLog [get]
func (leadApi *LeadApi) GetRunningLog(c *gin.Context) {
	runningId, logs := leadService.RunningInfo()
	response.OkWithDetailed(gin.H{
		"RunningTaskId": runningId,
		"Log":           logs,
	}, "获取成功", c)
}

// GenKeywords AI 生成关键词
// @Tags Lead
// @Summary 依据行业生成搜索关键词
// @Security ApiKeyAuth
// @Router /lead/genKeywords [post]
func (leadApi *LeadApi) GenKeywords(c *gin.Context) {
	var req struct {
		Industry string `json:"Industry"`
		Type     int    `json:"Type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if req.Industry == "" {
		response.FailWithMessage("请填写行业", c)
		return
	}

	keywords, err := leadService.GenKeywords(req.Industry, req.Type)
	if err != nil {
		global.GVA_LOG.Error("生成关键词失败!", zap.Error(err))
		response.FailWithMessage("生成失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(keywords, "生成成功", c)
}

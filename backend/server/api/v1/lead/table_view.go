package lead

import (

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

type tableViewReq struct {
	TableKey string `json:"tableKey"`
	Name     string `json:"name"`
	Payload  string `json:"payload"`
}

// TableViewList 视图列表
func (leadApi *LeadApi) TableViewList(c *gin.Context) {
	rows, err := leadService.TableViewList(c.Query("tableKey"))
	if err != nil {
		response.FailWithMessage("获取失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(rows, "获取成功", c)
}

// TableViewSave 保存视图
func (leadApi *LeadApi) TableViewSave(c *gin.Context) {
	var req tableViewReq

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if req.TableKey == "" || req.Name == "" {
		response.FailWithMessage("tableKey 和 name 不能为空", c)
		return
	}

	// "__" 开头是系统保留名：只允许系统自己的 __columns，其它一律拒绝
	if len(req.Name) >= 2 && req.Name[0] == '_' && req.Name[1] == '_' && req.Name != "__columns" {
		response.FailWithMessage("视图名称不能以 __ 开头", c)
		return
	}

	err := leadService.TableViewSave(req.TableKey, req.Name, req.Payload)
	if err != nil {
		global.GVA_LOG.Error("保存视图失败:" + err.Error())
		response.FailWithMessage("保存失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("保存成功", c)
}

// TableViewDelete 删除视图
func (leadApi *LeadApi) TableViewDelete(c *gin.Context) {
	if err := leadService.TableViewDelete(c.Query("tableKey"), c.Query("name")); err != nil {
		response.FailWithMessage("删除失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

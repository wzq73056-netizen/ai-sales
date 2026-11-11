package kb

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetKnowledgeStats 知识库分片数量统计
// @Tags Knowledge
// @Summary 知识库分片数量统计
// @Security ApiKeyAuth
// @Produce application/json
// @Success 200 {object} response.Response{data=[]kb.KnowledgeStat}
// @Router /kb/getKnowledgeStats [get]
func (kbApi *KnowledgeApi) GetKnowledgeStats(c *gin.Context) {
	stats, err := kbService.GetKnowledgeStats(c.Request.Context())
	if err != nil {
		global.GVA_LOG.Error("统计失败!", zap.Error(err))
		response.FailWithMessage("统计失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(stats, "获取成功", c)
}

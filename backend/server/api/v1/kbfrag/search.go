package kbfrag

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SearchRequest 检索入参
type SearchRequest struct {
	KnowledgeId int64   `json:"knowledgeId" form:"knowledgeId"`
	Query       string  `json:"query" form:"query"`
	TopK        int     `json:"topK" form:"topK"`
	MinScore    float64 `json:"minScore" form:"minScore"`
}

// SearchFragments 在知识库内做 BM25 检索
// @Tags KnowledgeFragment
// @Summary 检索知识库分片
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{data=[]kbfrag.SearchHit}
// @Router /kbfrag/searchFragments [post]
func (kbfragApi *KnowledgeFragmentApi) SearchFragments(c *gin.Context) {
	var req SearchRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if err2 := c.ShouldBindQuery(&req); err2 != nil {
			response.FailWithMessage(err.Error(), c)
			return
		}
	}

	if req.Query == "" {
		response.FailWithMessage("检索内容不能为空", c)
		return
	}

	hits, err := kbfragService.SearchFragments(c.Request.Context(), req.KnowledgeId, req.Query, req.TopK, req.MinScore)
	if err != nil {
		global.GVA_LOG.Error("检索失败!", zap.Error(err))
		response.FailWithMessage("检索失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(hits, "检索成功", c)
}

package kbfrag

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ChatRequest RAG 问答入参
type ChatRequest struct {
	KnowledgeId int64  `json:"knowledgeId" form:"knowledgeId"`
	Question    string `json:"question" form:"question"`
	TopK        int    `json:"topK" form:"topK"`
}

// Chat 检索增强问答
// @Tags KnowledgeFragment
// @Summary 知识库问答（RAG）
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{data=kbfrag.ChatResult}
// @Router /kbfrag/chat [post]
func (kbfragApi *KnowledgeFragmentApi) Chat(c *gin.Context) {
	var req ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if req.Question == "" {
		response.FailWithMessage("问题不能为空", c)
		return
	}

	result, err := kbfragService.Chat(c.Request.Context(), req.KnowledgeId, req.Question, req.TopK)
	if err != nil {
		global.GVA_LOG.Error("问答失败!", zap.Error(err))
		response.FailWithMessage("问答失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "问答成功", c)
}

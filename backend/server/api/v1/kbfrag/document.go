package kbfrag

import (
	"io"
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetDocumentList 文档列表
// @Tags KnowledgeDocument
// @Summary 知识库文档列表
// @Security ApiKeyAuth
// @Produce application/json
// @Router /kbfrag/getDocumentList [get]
func (kbfragApi *KnowledgeFragmentApi) GetDocumentList(c *gin.Context) {
	knowledgeId, _ := strconv.ParseInt(c.Query("knowledgeId"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	list, total, err := kbfragService.GetDocumentList(c.Request.Context(), knowledgeId, page, pageSize)
	if err != nil {
		global.GVA_LOG.Error("获取文档失败!", zap.Error(err))
		response.FailWithMessage("获取文档失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

// UploadDocument 上传文档并自动切片
// @Tags KnowledgeDocument
// @Summary 上传知识库文档
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce application/json
// @Router /kbfrag/uploadDocument [post]
func (kbfragApi *KnowledgeFragmentApi) UploadDocument(c *gin.Context) {
	knowledgeId, _ := strconv.ParseInt(c.PostForm("knowledgeId"), 10, 64)
	if knowledgeId == 0 {
		response.FailWithMessage("缺少 knowledgeId", c)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.FailWithMessage("未收到文件:"+err.Error(), c)
		return
	}

	opened, err := file.Open()
	if err != nil {
		response.FailWithMessage("读取文件失败:"+err.Error(), c)
		return
	}
	defer opened.Close()

	content, err := io.ReadAll(opened)
	if err != nil {
		response.FailWithMessage("读取文件失败:"+err.Error(), c)
		return
	}

	doc, err := kbfragService.UploadDocument(c.Request.Context(), knowledgeId, file.Filename, content)
	if err != nil {
		global.GVA_LOG.Error("上传失败!", zap.Error(err))
		response.FailWithMessage("上传失败:"+err.Error(), c)
		return
	}

	response.OkWithDetailed(doc, "上传成功", c)
}

// DeleteDocument 删除文档
// @Tags KnowledgeDocument
// @Summary 删除知识库文档
// @Security ApiKeyAuth
// @Produce application/json
// @Router /kbfrag/deleteDocument [delete]
func (kbfragApi *KnowledgeFragmentApi) DeleteDocument(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Query("Id"), 10, 64)

	if err := kbfragService.DeleteDocument(c.Request.Context(), id); err != nil {
		global.GVA_LOG.Error("删除文档失败!", zap.Error(err))
		response.FailWithMessage("删除失败:"+err.Error(), c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

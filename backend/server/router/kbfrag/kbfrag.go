package kbfrag

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type KnowledgeFragmentRouter struct {}

// InitKnowledgeFragmentRouter 初始化 知识库分片 路由信息
func (s *KnowledgeFragmentRouter) InitKnowledgeFragmentRouter(Router *gin.RouterGroup,PublicRouter *gin.RouterGroup) {
	kbfragRouter := Router.Group("kbfrag").Use(middleware.OperationRecord())
	kbfragRouterWithoutRecord := Router.Group("kbfrag")
	kbfragRouterWithoutAuth := PublicRouter.Group("kbfrag")
	{
		kbfragRouter.POST("createKnowledgeFragment", kbfragApi.CreateKnowledgeFragment)   // 新建知识库分片
		kbfragRouter.DELETE("deleteKnowledgeFragment", kbfragApi.DeleteKnowledgeFragment) // 删除知识库分片
		kbfragRouter.DELETE("deleteKnowledgeFragmentByIds", kbfragApi.DeleteKnowledgeFragmentByIds) // 批量删除知识库分片
		kbfragRouter.PUT("updateKnowledgeFragment", kbfragApi.UpdateKnowledgeFragment)    // 更新知识库分片
	}
	{
		kbfragRouterWithoutRecord.GET("findKnowledgeFragment", kbfragApi.FindKnowledgeFragment)        // 根据ID获取知识库分片
		kbfragRouterWithoutRecord.GET("getKnowledgeFragmentList", kbfragApi.GetKnowledgeFragmentList)  // 获取知识库分片列表
	kbfragRouterWithoutRecord.POST("searchFragments", kbfragApi.SearchFragments)                 // 检索知识库分片
	kbfragRouterWithoutRecord.GET("getDocumentList", kbfragApi.GetDocumentList)                    // 文档列表
	kbfragRouterWithoutRecord.POST("uploadDocument", kbfragApi.UploadDocument)                      // 上传文档
	kbfragRouterWithoutRecord.DELETE("deleteDocument", kbfragApi.DeleteDocument)                    // 删除文档
	kbfragRouterWithoutRecord.POST("chat", kbfragApi.Chat)                                        // 知识库问答（RAG）
	}
	{
	    kbfragRouterWithoutAuth.GET("getKnowledgeFragmentPublic", kbfragApi.GetKnowledgeFragmentPublic)  // 知识库分片开放接口
	}
}

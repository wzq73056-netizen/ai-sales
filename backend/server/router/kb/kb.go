package kb

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type KnowledgeRouter struct {}

// InitKnowledgeRouter 初始化 KnowledgeBase 路由信息
func (s *KnowledgeRouter) InitKnowledgeRouter(Router *gin.RouterGroup,PublicRouter *gin.RouterGroup) {
	kbRouter := Router.Group("kb").Use(middleware.OperationRecord())
	kbRouterWithoutRecord := Router.Group("kb")
	kbRouterWithoutAuth := PublicRouter.Group("kb")
	{
		kbRouter.POST("createKnowledge", kbApi.CreateKnowledge)   // 新建KnowledgeBase
		kbRouter.DELETE("deleteKnowledge", kbApi.DeleteKnowledge) // 删除KnowledgeBase
		kbRouter.DELETE("deleteKnowledgeByIds", kbApi.DeleteKnowledgeByIds) // 批量删除KnowledgeBase
		kbRouter.PUT("updateKnowledge", kbApi.UpdateKnowledge)    // 更新KnowledgeBase
	}
	{
		kbRouterWithoutRecord.GET("findKnowledge", kbApi.FindKnowledge)        // 根据ID获取KnowledgeBase
		kbRouterWithoutRecord.GET("getKnowledgeList", kbApi.GetKnowledgeList)  // 获取KnowledgeBase列表
	kbRouterWithoutRecord.GET("getKnowledgeStats", kbApi.GetKnowledgeStats)   // 知识库分片数量统计
	}
	{
	    kbRouterWithoutAuth.GET("getKnowledgePublic", kbApi.GetKnowledgePublic)  // KnowledgeBase开放接口
	}
}

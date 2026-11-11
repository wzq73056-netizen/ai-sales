package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router"
	"github.com/gin-gonic/gin"
)

// 占位方法，保证文件可以被正确加载，避免 go 空变量检测报错，请勿删除。
func holder(routers ...*gin.RouterGroup) {
	_ = routers
	_ = router.RouterGroupApp
}

func initBizRouter(routers ...*gin.RouterGroup) {
	privateGroup := routers[0]
	publicGroup := routers[1]

	holder(publicGroup, privateGroup)
	{
		kbRouter := router.RouterGroupApp.Kb
		kbRouter.InitKnowledgeRouter(privateGroup, publicGroup)
	}
	{
		kbfragRouter := router.RouterGroupApp.Kbfrag
		kbfragRouter.InitKnowledgeFragmentRouter(privateGroup, publicGroup)
	}

	{
		leadRouter := router.RouterGroupApp.Lead
		leadRouter.InitLeadRouter(privateGroup, publicGroup)
	}
}

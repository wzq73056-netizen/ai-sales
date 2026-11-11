package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router/example"
	"github.com/flipped-aurora/gin-vue-admin/server/router/kb"
	"github.com/flipped-aurora/gin-vue-admin/server/router/kbfrag"
	"github.com/flipped-aurora/gin-vue-admin/server/router/lead"
	"github.com/flipped-aurora/gin-vue-admin/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	System    system.RouterGroup
	Example   example.RouterGroup
	Kb        kb.RouterGroup
	Kbfrag    kbfrag.RouterGroup
	Lead      lead.RouterGroup
}

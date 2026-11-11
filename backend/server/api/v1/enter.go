package v1

import (
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/example"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/kb"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/kbfrag"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/lead"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/system"
)

var ApiGroupApp = new(ApiGroup)

type ApiGroup struct {
	SystemApiGroup    system.ApiGroup
	ExampleApiGroup   example.ApiGroup
	KbApiGroup        kb.ApiGroup
	KbfragApiGroup    kbfrag.ApiGroup
	LeadApiGroup      lead.ApiGroup
}

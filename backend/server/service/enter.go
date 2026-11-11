package service

import (
	"github.com/flipped-aurora/gin-vue-admin/server/service/example"
	"github.com/flipped-aurora/gin-vue-admin/server/service/kb"
	"github.com/flipped-aurora/gin-vue-admin/server/service/kbfrag"
	"github.com/flipped-aurora/gin-vue-admin/server/service/lead"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
)

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	SystemServiceGroup    system.ServiceGroup
	ExampleServiceGroup   example.ServiceGroup
	KbServiceGroup        kb.ServiceGroup
	KbfragServiceGroup    kbfrag.ServiceGroup
	LeadServiceGroup      lead.ServiceGroup
}

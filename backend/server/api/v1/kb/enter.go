package kb

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	KnowledgeApi
}

var kbService = service.ServiceGroupApp.KbServiceGroup.KnowledgeService

package kbfrag

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	KnowledgeFragmentApi
}

var kbfragService = service.ServiceGroupApp.KbfragServiceGroup.KnowledgeFragmentService

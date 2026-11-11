package kbfrag

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	KnowledgeFragmentRouter
}

var kbfragApi = api.ApiGroupApp.KbfragApiGroup.KnowledgeFragmentApi

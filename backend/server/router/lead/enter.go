package lead

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	LeadRouter
}

var leadApi = api.ApiGroupApp.LeadApiGroup.LeadApi

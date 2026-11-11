package lead

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	LeadApi
}

var leadService = service.ServiceGroupApp.LeadServiceGroup.LeadService

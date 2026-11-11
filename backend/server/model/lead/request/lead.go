
package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	
)

type LeadSearch struct{
    request.PageInfo
    Sort  string `json:"sort" form:"sort"`
    Order string `json:"order" form:"order"`
	// 手工新增：按跟进状态过滤
	Status int `json:"status" form:"status"`
	Source string `json:"source" form:"source"`
	Grade  string `json:"grade" form:"grade"`
	Side   string `json:"side" form:"side"`
}

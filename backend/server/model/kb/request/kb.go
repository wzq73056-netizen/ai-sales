package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type KnowledgeSearch struct {
	request.PageInfo
	Sort  string `json:"sort" form:"sort"`
	Order string `json:"order" form:"order"`
	// 手工新增：按名称模糊搜索
	Name string `json:"name" form:"name"`
}

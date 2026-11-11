package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type KnowledgeFragmentSearch struct {
	request.PageInfo
	Sort  string `json:"sort" form:"sort"`
	Order string `json:"order" form:"order"`
	// 手工新增：按所属知识库过滤（生成器不会生成业务字段）
	KnowledgeId int64 `json:"knowledgeId" form:"knowledgeId"`
}

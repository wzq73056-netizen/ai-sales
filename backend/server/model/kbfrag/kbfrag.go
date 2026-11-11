
// 自动生成模板KnowledgeFragment
package kbfrag
import (
)

// 知识库分片 结构体  KnowledgeFragment
type KnowledgeFragment struct {
  Id  *int64 `json:"Id" form:"Id" gorm:"primarykey;column:id;"`
  KnowledgeId  *int64 `json:"KnowledgeId" form:"KnowledgeId" gorm:"column:knowledge_id;"`  //所属知识库（老表靠 doc_id 隐式关联，这里显式化）
  DocumentId  *int64 `json:"DocumentId" form:"DocumentId" gorm:"column:document_id;"`  //来源文档
  Idx  *int64 `json:"Idx" form:"Idx" gorm:"column:idx;"`  //片段序号
  DocId  *string `json:"DocId" form:"DocId" gorm:"column:doc_id;"`  //文档ID
  Content  *string `json:"Content" form:"Content" gorm:"column:content;"`  //片段内容
  CreatedAt  *string `json:"CreatedAt" form:"CreatedAt" gorm:"column:created_at;"`
  UpdatedAt  *string `json:"UpdatedAt" form:"UpdatedAt" gorm:"column:updated_at;"`
  DeletedAt  *string `json:"DeletedAt" form:"DeletedAt" gorm:"column:deleted_at;"`
}


// TableName 知识库分片 KnowledgeFragment自定义表名 ai_knowledge_fragment
func (KnowledgeFragment) TableName() string {
    return "ai_knowledge_fragment"
}






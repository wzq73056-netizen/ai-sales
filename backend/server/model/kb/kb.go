
// 自动生成模板Knowledge
package kb
import (
)

// KnowledgeBase 结构体  Knowledge
type Knowledge struct {
  Id  *int64 `json:"Id" form:"Id" gorm:"primarykey;column:id;"`
  UserId  *int64 `json:"UserId" form:"UserId" gorm:"column:user_id;"`  //所有者用户ID
  Name  *string `json:"Name" form:"Name" gorm:"column:name;"`  //知识库名称
  Description  *string `json:"Description" form:"Description" gorm:"column:description;"`  //描述
  ChunkSeparator  *string `json:"ChunkSeparator" form:"ChunkSeparator" gorm:"column:chunk_separator;"`  //分句标识符
  ChunkSize  *int64 `json:"ChunkSize" form:"ChunkSize" gorm:"column:chunk_size;"`  //分段预估长度
  OverlapSize  *int64 `json:"OverlapSize" form:"OverlapSize" gorm:"column:overlap_size;"`  //分段重叠长度
  RetrieveLimit  *int64 `json:"RetrieveLimit" form:"RetrieveLimit" gorm:"column:retrieve_limit;"`  //检索条数
  RerankMinScore  *float64 `json:"RerankMinScore" form:"RerankMinScore" gorm:"column:rerank_min_score;"`  //相似度阈值
  EmbeddingModel  *string `json:"EmbeddingModel" form:"EmbeddingModel" gorm:"column:embedding_model;"`  //向量模型
  VectorStore  *string `json:"VectorStore" form:"VectorStore" gorm:"column:vector_store;"`  //向量存储类型
  Share  *int64 `json:"Share" form:"Share" gorm:"column:share;"`  //是否公开 0否 1是
  Status  *int64 `json:"Status" form:"Status" gorm:"column:status;"`  //状态 1正常 0停用
  RequestCount  *int64 `json:"RequestCount" form:"RequestCount" gorm:"column:request_count;"`  //调用次数
  LegacyTable  *string `json:"LegacyTable" form:"LegacyTable" gorm:"column:legacy_table;"`  //来源表
  LegacyId  *int64 `json:"LegacyId" form:"LegacyId" gorm:"column:legacy_id;"`  //来源主键
  CreatedAt  *string `json:"CreatedAt" form:"CreatedAt" gorm:"column:created_at;"`
  UpdatedAt  *string `json:"UpdatedAt" form:"UpdatedAt" gorm:"column:updated_at;"`
  DeletedAt  *string `json:"DeletedAt" form:"DeletedAt" gorm:"column:deleted_at;"`  //软删除时间
}


// TableName KnowledgeBase Knowledge自定义表名 ai_knowledge
func (Knowledge) TableName() string {
    return "ai_knowledge"
}






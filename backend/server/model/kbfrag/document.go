package kbfrag

// KnowledgeDocument 知识库文档
type KnowledgeDocument struct {
	Id          int64  `json:"Id" gorm:"primarykey;column:id;"`
	KnowledgeId int64  `json:"KnowledgeId" gorm:"column:knowledge_id;"`
	Name        string `json:"Name" gorm:"column:name;"`
	Format      string `json:"Format" gorm:"column:format;"`
	Size        int64  `json:"Size" gorm:"column:size;"`
	ChunkCount  int    `json:"ChunkCount" gorm:"column:chunk_count;"`
	Status      int    `json:"Status" gorm:"column:status;"`
	StoragePath string `json:"StoragePath" gorm:"column:storage_path;"`
	CreatedAt   string `json:"CreatedAt" gorm:"column:created_at;"`
	UpdatedAt   string `json:"UpdatedAt" gorm:"column:updated_at;"`
}

// TableName 表名
func (KnowledgeDocument) TableName() string {
	return "ai_knowledge_document"
}

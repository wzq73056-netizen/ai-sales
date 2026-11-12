package lead

// TableView 保存的表格视图（筛选 + 列 + 排序），按表格和名称唯一
//
// 放在 model 下是为了能被 initialize/gorm_biz.go 的 AutoMigrate 注册，
// 全新安装时自动建表；放在 service 下时无法被迁移逻辑引用。
type TableView struct {
	Id        int    `gorm:"primarykey" json:"Id"`
	TableKey  string `json:"TableKey" gorm:"column:table_key;size:64;index"`
	Name      string `json:"Name" gorm:"column:name;size:80"`
	Payload   string `json:"Payload" gorm:"column:payload;type:text"`
	CreatedAt int64  `json:"CreatedAt" gorm:"column:created_at"`
	UpdatedAt int64  `json:"UpdatedAt" gorm:"column:updated_at"`
}

func (TableView) TableName() string {
	return "ai_table_view"
}

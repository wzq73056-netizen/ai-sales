package lead

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// TableView 保存的表格视图（筛选 + 列 + 排序），按表格和名称唯一
type TableView struct {
	Id       int    `gorm:"primarykey" json:"Id"`
	TableKey string `json:"TableKey" gorm:"column:table_key;size:64;index"`
	Name     string `json:"Name" gorm:"column:name;size:80"`
	Payload  string `json:"Payload" gorm:"column:payload;type:text"`
	CreatedAt int64 `json:"CreatedAt" gorm:"column:created_at"`
	UpdatedAt int64 `json:"UpdatedAt" gorm:"column:updated_at"`
}

func (TableView) TableName() string {
	return "ai_table_view"
}

// TableViewList 返回某张表已保存的视图
func (leadService *LeadService) TableViewList(tableKey string) ([]TableView, error) {
	var rows []TableView

	err := global.GVA_DB.Where("table_key = ?", tableKey).Order("id asc").Find(&rows).Error

	return rows, err
}

// TableViewSave 新增或覆盖同名视图
func (leadService *LeadService) TableViewSave(tableKey, name, payload string) error {
	var exist TableView

	db := global.GVA_DB.Where("table_key = ? AND name = ?", tableKey, name)

	if err := db.First(&exist).Error; err == nil {
		return global.GVA_DB.Model(&exist).Updates(map[string]any{
			"payload":    payload,
			"updated_at": time.Now().Unix(),
		}).Error
	}

	now := time.Now().Unix()

	return global.GVA_DB.Create(&TableView{
		TableKey:  tableKey,
		Name:      name,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error
}

// TableViewDelete 按名称删除视图
func (leadService *LeadService) TableViewDelete(tableKey, name string) error {
	return global.GVA_DB.Where("table_key = ? AND name = ?", tableKey, name).Delete(&TableView{}).Error
}

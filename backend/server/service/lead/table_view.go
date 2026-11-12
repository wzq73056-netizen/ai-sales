package lead

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

func nowUnix() int64 { return time.Now().Unix() }

// TableViewList 返回某张表已保存的视图
func (leadService *LeadService) TableViewList(tableKey string) ([]leadModel.TableView, error) {
	var rows []leadModel.TableView

	err := global.GVA_DB.Where("table_key = ?", tableKey).Order("id asc").Find(&rows).Error

	return rows, err
}

// TableViewSave 新增或覆盖同名视图
func (leadService *LeadService) TableViewSave(tableKey, name, payload string) error {
	var exist leadModel.TableView

	db := global.GVA_DB.Where("table_key = ? AND name = ?", tableKey, name)

	if err := db.First(&exist).Error; err == nil {
		return global.GVA_DB.Model(&exist).Updates(map[string]any{
			"payload":    payload,
			"updated_at": nowUnix(),
		}).Error
	}

	now := nowUnix()

	return global.GVA_DB.Create(&leadModel.TableView{
		TableKey:  tableKey,
		Name:      name,
		Payload:   payload,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error
}

// TableViewDelete 按名称删除视图
func (leadService *LeadService) TableViewDelete(tableKey, name string) error {
	return global.GVA_DB.Where("table_key = ? AND name = ?", tableKey, name).Delete(&leadModel.TableView{}).Error
}
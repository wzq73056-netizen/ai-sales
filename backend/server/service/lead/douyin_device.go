package lead

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// GetDouyinDevices 账号列表（platform 为空返回全部）
func (leadService *LeadService) GetDouyinDevices(platform string) ([]leadModel.DouyinDevice, error) {
	var list []leadModel.DouyinDevice

	query := global.GVA_DB.Order("id asc")

	if platform != "" {
		query = query.Where("platform = ?", platform)
	}

	err := query.Find(&list).Error

	return list, err
}

// SaveDouyinDevice 新增或更新账号
func (leadService *LeadService) SaveDouyinDevice(item leadModel.DouyinDevice) (leadModel.DouyinDevice, error) {
	if item.Id == 0 {
		err := global.GVA_DB.Create(&item).Error

		return item, err
	}

	updates := map[string]any{
		"platform":     item.Platform,
		"name":         item.Name,
		"douyin_id":    item.DouyinId,
		"profile_dir":  item.ProfileDir,
		"browser_path": item.BrowserPath,
		"browser_name": item.BrowserName,
		"cdp_port":     item.CdpPort,
		"status":       item.Status,
	}

	return item, global.GVA_DB.Model(&leadModel.DouyinDevice{}).Where("id = ?", item.Id).Updates(updates).Error
}

// DeleteDouyinDevice 删除账号
func (leadService *LeadService) DeleteDouyinDevice(id uint) error {
	return global.GVA_DB.Delete(&leadModel.DouyinDevice{}, id).Error
}

// GetDouyinWhitelist 白名单（可按账号过滤）
func (leadService *LeadService) GetDouyinWhitelist(deviceId uint, platform string) ([]leadModel.DouyinWhitelist, error) {
	var list []leadModel.DouyinWhitelist

	query := global.GVA_DB.Order("id asc")

	if deviceId > 0 {
		query = query.Where("device_id = ?", deviceId)
	}

	if platform != "" {
		query = query.Where("platform = ?", platform)
	}

	err := query.Find(&list).Error

	return list, err
}

// SaveDouyinWhitelist 新增或更新白名单
func (leadService *LeadService) SaveDouyinWhitelist(item leadModel.DouyinWhitelist) (leadModel.DouyinWhitelist, error) {
	if item.Id == 0 {
		err := global.GVA_DB.Create(&item).Error

		return item, err
	}

	updates := map[string]any{
		"device_id":   item.DeviceId,
		"nickname":    item.Nickname,
		"douyin_id":   item.DouyinId,
		"profile_url": item.ProfileUrl,
		"enabled":     item.Enabled,
	}

	return item, global.GVA_DB.Model(&leadModel.DouyinWhitelist{}).Where("id = ?", item.Id).Updates(updates).Error
}

// DeleteDouyinWhitelist 删除白名单
func (leadService *LeadService) DeleteDouyinWhitelist(id uint) error {
	return global.GVA_DB.Delete(&leadModel.DouyinWhitelist{}, id).Error
}

// TouchDouyinDevice 记录账号最近一次心跳
func (leadService *LeadService) TouchDouyinDevice(id uint) error {
	return global.GVA_DB.Model(&leadModel.DouyinDevice{}).Where("id = ?", id).
		Update("last_seen", time.Now()).Error
}

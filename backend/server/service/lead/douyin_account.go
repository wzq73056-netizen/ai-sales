package lead

import (
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// SaveDouyinAccounts 批量写入账号线索（按 profile_url 去重，已存在则更新分数/粉丝）
func (leadService *LeadService) SaveDouyinAccounts(list []leadModel.DouyinAccount) (int, error) {
	created := 0
	now := time.Now().Format("2006-01-02 15:04:05")

	for i := range list {
		item := list[i]
		if strings.TrimSpace(item.ProfileUrl) == "" {
			continue
		}

		var exist leadModel.DouyinAccount
		err := global.GVA_DB.Where("profile_url = ?", item.ProfileUrl).First(&exist).Error

		if err == nil {
			global.GVA_DB.Model(&exist).Updates(map[string]any{
				"fans":          item.Fans,
				"score":         item.Score,
				"is_competitor": item.IsCompetitor,
				"bio":           item.Bio,
				"updated_at":    now,
			})
			continue
		}

		item.CreatedAt = &now
		item.UpdatedAt = &now
		if e := global.GVA_DB.Create(&item).Error; e == nil {
			created++
		}
	}

	return created, nil
}

// GetDouyinAccountList 账号线索列表
// sortBy: fans(默认) / score / id
func (leadService *LeadService) GetDouyinAccountList(page, pageSize int, keyword string, onlyCompetitor int, sortBy string) ([]leadModel.DouyinAccount, int64, error) {
	var list []leadModel.DouyinAccount
	var total int64

	db := global.GVA_DB.Model(&leadModel.DouyinAccount{})

	if keyword != "" {
		db = db.Where("keyword LIKE ? OR nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if onlyCompetitor == 1 {
		db = db.Where("is_competitor = 1")
	} else if onlyCompetitor == 2 {
		db = db.Where("is_competitor = 0")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	order := "fans DESC"
	switch sortBy {
	case "score":
		order = "score DESC"
	case "id":
		order = "id DESC"
	}

	err := db.Order(order).Limit(pageSize).Offset((page - 1) * pageSize).Find(&list).Error
	return list, total, err
}

// DeleteDouyinAccount 删除账号线索
func (leadService *LeadService) DeleteDouyinAccount(id int64) error {
	if id == 0 {
		return fmt.Errorf("缺少 id")
	}
	return global.GVA_DB.Delete(&leadModel.DouyinAccount{}, "id = ?", id).Error
}

// UpdateDouyinAccount 更新状态与跟进备注
func (leadService *LeadService) UpdateDouyinAccount(id int64, status int, remark string) error {
	updates := map[string]any{"status": status}
	if remark != "" {
		updates["remark"] = remark
		updates["follow_at"] = time.Now().Format("2006-01-02 15:04:05")
	}
	return global.GVA_DB.Model(&leadModel.DouyinAccount{}).Where("id = ?", id).Updates(updates).Error
}


package kb

import (
	"context"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/kb"
	kbfrag "github.com/flipped-aurora/gin-vue-admin/server/model/kbfrag"
	kbReq "github.com/flipped-aurora/gin-vue-admin/server/model/kb/request"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type KnowledgeService struct {}
// CreateKnowledge 创建KnowledgeBase记录
// Author [yourname](https://github.com/yourname)
func (kbService *KnowledgeService) CreateKnowledge(ctx context.Context, kb *kb.Knowledge) (err error) {
	err = global.GVA_DB.WithContext(ctx).Create(kb).Error
	return err
}

// DeleteKnowledge 删除KnowledgeBase记录（级联删除其下所有分片）
func (kbService *KnowledgeService)DeleteKnowledge(ctx context.Context, Id string) (err error) {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("knowledge_id = ?", Id).Delete(&kbfrag.KnowledgeFragment{}).Error; e != nil {
			return e
		}
		return tx.Delete(&kb.Knowledge{},"id = ?",Id).Error
	})
}

// DeleteKnowledgeByIds 批量删除KnowledgeBase记录（级联删除分片）
func (kbService *KnowledgeService)DeleteKnowledgeByIds(ctx context.Context, Ids []string) (err error) {
	return global.GVA_DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("knowledge_id in ?", Ids).Delete(&kbfrag.KnowledgeFragment{}).Error; e != nil {
			return e
		}
		return tx.Delete(&[]kb.Knowledge{},"id in ?",Ids).Error
	})
}

// UpdateKnowledge 更新KnowledgeBase记录
// Author [yourname](https://github.com/yourname)
func (kbService *KnowledgeService)UpdateKnowledge(ctx context.Context, data kb.Knowledge) (err error) {
	err = global.GVA_DB.WithContext(ctx).Model(&kb.Knowledge{}).Where("id = ?",data.Id).Updates(&data).Error
	return err
}

// GetKnowledge 根据Id获取KnowledgeBase记录
// Author [yourname](https://github.com/yourname)
func (kbService *KnowledgeService)GetKnowledge(ctx context.Context, Id string) (data kb.Knowledge, err error) {
	err = global.GVA_DB.WithContext(ctx).Where("id = ?", Id).First(&data).Error
	return
}
// GetKnowledgeInfoList 分页获取KnowledgeBase记录
// Author [yourname](https://github.com/yourname)
func (kbService *KnowledgeService)GetKnowledgeInfoList(ctx context.Context, info kbReq.KnowledgeSearch) (list []kb.Knowledge, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
    // 创建db
	db := global.GVA_DB.WithContext(ctx).Model(&kb.Knowledge{})
    var kbs []kb.Knowledge
    // 手工新增：名称模糊搜索
    if info.Name != "" {
        db = db.Where("name LIKE ?", "%"+info.Name+"%")
    }
    
	err = db.Count(&total).Error
	if err!=nil {
    	return
    }
        orderMap := make(map[string]bool)
         	orderMap["id"] = true
       if orderMap[info.Sort] {
          db = db.Order(clause.OrderByColumn{
             Column: clause.Column{Name: info.Sort},
             Desc: info.Order == "descending",
          })
       }

	if limit != 0 {
       db = db.Limit(limit).Offset(offset)
    }

	err = db.Find(&kbs).Error
	return  kbs, total, err
}
func (kbService *KnowledgeService)GetKnowledgePublic(ctx context.Context) {
    // 此方法为获取数据源定义的数据
    // 请自行实现
}

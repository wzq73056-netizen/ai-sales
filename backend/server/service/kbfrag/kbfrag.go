
package kbfrag

import (
	"context"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/kbfrag"
    kbfragReq "github.com/flipped-aurora/gin-vue-admin/server/model/kbfrag/request"
	"gorm.io/gorm/clause"
)

type KnowledgeFragmentService struct {}
// CreateKnowledgeFragment 创建知识库分片记录
// Author [yourname](https://github.com/yourname)
func (kbfragService *KnowledgeFragmentService) CreateKnowledgeFragment(ctx context.Context, kbfrag *kbfrag.KnowledgeFragment) (err error) {
	err = global.GVA_DB.WithContext(ctx).Create(kbfrag).Error
	return err
}

// DeleteKnowledgeFragment 删除知识库分片记录
// Author [yourname](https://github.com/yourname)
func (kbfragService *KnowledgeFragmentService)DeleteKnowledgeFragment(ctx context.Context, Id string) (err error) {
	err = global.GVA_DB.WithContext(ctx).Delete(&kbfrag.KnowledgeFragment{},"id = ?",Id).Error
	return err
}

// DeleteKnowledgeFragmentByIds 批量删除知识库分片记录
// Author [yourname](https://github.com/yourname)
func (kbfragService *KnowledgeFragmentService)DeleteKnowledgeFragmentByIds(ctx context.Context, Ids []string) (err error) {
	err = global.GVA_DB.WithContext(ctx).Delete(&[]kbfrag.KnowledgeFragment{},"id in ?",Ids).Error
	return err
}

// UpdateKnowledgeFragment 更新知识库分片记录
// Author [yourname](https://github.com/yourname)
func (kbfragService *KnowledgeFragmentService)UpdateKnowledgeFragment(ctx context.Context, data kbfrag.KnowledgeFragment) (err error) {
	err = global.GVA_DB.WithContext(ctx).Model(&kbfrag.KnowledgeFragment{}).Where("id = ?",data.Id).Updates(&data).Error
	return err
}

// GetKnowledgeFragment 根据Id获取知识库分片记录
// Author [yourname](https://github.com/yourname)
func (kbfragService *KnowledgeFragmentService)GetKnowledgeFragment(ctx context.Context, Id string) (data kbfrag.KnowledgeFragment, err error) {
	err = global.GVA_DB.WithContext(ctx).Where("id = ?", Id).First(&data).Error
	return
}
// GetKnowledgeFragmentInfoList 分页获取知识库分片记录
// Author [yourname](https://github.com/yourname)
func (kbfragService *KnowledgeFragmentService)GetKnowledgeFragmentInfoList(ctx context.Context, info kbfragReq.KnowledgeFragmentSearch) (list []kbfrag.KnowledgeFragment, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
    // 创建db
	db := global.GVA_DB.WithContext(ctx).Model(&kbfrag.KnowledgeFragment{})
    var kbfrags []kbfrag.KnowledgeFragment
    // 手工新增：按知识库过滤（父子表关联）
    if info.KnowledgeId != 0 {
        db = db.Where("knowledge_id = ?", info.KnowledgeId)
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

	err = db.Find(&kbfrags).Error
	return  kbfrags, total, err
}
func (kbfragService *KnowledgeFragmentService)GetKnowledgeFragmentPublic(ctx context.Context) {
    // 此方法为获取数据源定义的数据
    // 请自行实现
}


package lead

import (
	"context"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/lead"
    leadReq "github.com/flipped-aurora/gin-vue-admin/server/model/lead/request"
	"gorm.io/gorm/clause"
)

type LeadService struct {}
// CreateLead 创建获客线索记录
// Author [yourname](https://github.com/yourname)
func (leadService *LeadService) CreateLead(ctx context.Context, lead *lead.Lead) (err error) {
	// 入库前先分级，销售只看 S/A
	grade, reason, ready := GradeLead(lead)
	lead.Grade = grade
	lead.GradeReason = reason

	if ready {
		lead.ContactReady = 1
	}
	// phone 有唯一索引：空值要存 NULL，否则多条空字符串会撞 uk_phone
	if lead.Phone != nil && strings.TrimSpace(*lead.Phone) == "" {
		lead.Phone = nil
	}

	// 生成器模型全是指针字段，NULL 会撞 NOT NULL 约束；这里兜底补空值
	empty := ""

	for _, field := range []**string{
		&lead.Avatar, &lead.Company, &lead.ContactName, &lead.City, &lead.Industry,
		&lead.Source, &lead.Keyword, &lead.Remark, &lead.Wechat,
		&lead.ExtractInfo, &lead.VideoTitle, &lead.SourceUrl,
	} {
		if *field == nil {
			*field = &empty
		}
	}

	now := time.Now().Format("2006-01-02 15:04:05.000")
	if lead.CreatedAt == nil {
		lead.CreatedAt = &now
	}
	if lead.UpdatedAt == nil {
		lead.UpdatedAt = &now
	}
	err = global.GVA_DB.WithContext(ctx).Create(lead).Error
	return err
}

// DeleteLead 删除获客线索记录
// Author [yourname](https://github.com/yourname)
func (leadService *LeadService)DeleteLead(ctx context.Context, Id string) (err error) {
	err = global.GVA_DB.WithContext(ctx).Delete(&lead.Lead{},"id = ?",Id).Error
	return err
}

// DeleteLeadByIds 批量删除获客线索记录
// Author [yourname](https://github.com/yourname)
func (leadService *LeadService)DeleteLeadByIds(ctx context.Context, Ids []string) (err error) {
	err = global.GVA_DB.WithContext(ctx).Delete(&[]lead.Lead{},"id in ?",Ids).Error
	return err
}

// UpdateLead 更新获客线索记录
// Author [yourname](https://github.com/yourname)
func (leadService *LeadService)UpdateLead(ctx context.Context, data lead.Lead) (err error) {
	err = global.GVA_DB.WithContext(ctx).Model(&lead.Lead{}).Where("id = ?",data.Id).Updates(&data).Error
	return err
}

// GetLead 根据Id获取获客线索记录
// Author [yourname](https://github.com/yourname)
func (leadService *LeadService)GetLead(ctx context.Context, Id string) (data lead.Lead, err error) {
	err = global.GVA_DB.WithContext(ctx).Where("id = ?", Id).First(&data).Error
	return
}
// GetLeadInfoList 分页获取获客线索记录
// Author [yourname](https://github.com/yourname)
func (leadService *LeadService)GetLeadInfoList(ctx context.Context, info leadReq.LeadSearch) (list []lead.Lead, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
    // 创建db
	db := global.GVA_DB.WithContext(ctx).Model(&lead.Lead{})
    var leads []lead.Lead
    // 手工新增：状态筛选
    if info.Source != "" {
        db = db.Where("source = ?", info.Source)
    }
    if info.Status > 0 {
        db = db.Where("status = ?", info.Status)
    }
    if info.Grade != "" {
        db = db.Where("grade = ?", info.Grade)
    }
    if info.Side != "" {
        db = db.Where("COALESCE(NULLIF(side,''),'buyer') = ?", info.Side)
    }
    // 如果有条件搜索 下方会自动创建搜索语句
    
	err = db.Count(&total).Error
	if err!=nil {
    	return
    }
        // 排序白名单：前端列名 → 数据库列
	orderMap := map[string]string{
		"id":          "id",
		"score":       "score",
		"created_at":  "created_at",
		"createdAt":   "created_at",
		"last_msg_at": "last_msg_at",
		"updated_at":  "updated_at",
		"grade":       "grade",
	}

	sortColumn, ok := orderMap[info.Sort]

	if ok {
		desc := info.Order == "descending" || info.Order == "desc" || info.Order == ""
		db = db.Order(clause.OrderByColumn{
			Column: clause.Column{Name: sortColumn},
			Desc:   desc,
		})
	}

	if limit != 0 {
       db = db.Limit(limit).Offset(offset)
    }

	err = db.Find(&leads).Error
	return  leads, total, err
}
func (leadService *LeadService)GetLeadPublic(ctx context.Context) {
    // 此方法为获取数据源定义的数据
    // 请自行实现
}

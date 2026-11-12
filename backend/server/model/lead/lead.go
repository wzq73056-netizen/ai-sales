
// 自动生成模板Lead
package lead
import (
	"strings"
)

// 获客线索 结构体  Lead
type Lead struct {
  Id  *int64 `json:"Id" form:"Id" gorm:"primarykey;column:id;"`
  Company  *string `json:"Company" form:"Company" gorm:"column:company;"`  //公司名称
  ContactName  *string `json:"ContactName" form:"ContactName" gorm:"column:contact_name;"`  //联系人
  Phone  *string `json:"Phone" form:"Phone" gorm:"column:phone;"`  //电话
  Wechat  *string `json:"Wechat" form:"Wechat" gorm:"column:wechat;"`  //微信
  Email  *string `json:"Email" form:"Email" gorm:"column:email;"`  //邮箱
  City  *string `json:"City" form:"City" gorm:"column:city;"`  //城市
  Industry  *string `json:"Industry" form:"Industry" gorm:"column:industry;"`  //行业
  Scale  *string `json:"Scale" form:"Scale" gorm:"column:scale;"`  //规模
  Source  *string `json:"Source" form:"Source" gorm:"column:source;"`  //来源：manual/scrape/import
  SourceUrl  *string `json:"SourceUrl" form:"SourceUrl" gorm:"column:source_url;type:text;"`  //来源链接
  Side  *string `json:"Side" form:"Side" gorm:"column:side;"`  //对方角色：buyer=买我们货的客户 / supplier=供货给我们的
  Keyword  *string `json:"Keyword" form:"Keyword" gorm:"column:keyword;"`  //采集关键词
  Score  *int64 `json:"Score" form:"Score" gorm:"column:score;"`  //线索评分 0-100
  Grade  string `json:"Grade" form:"Grade" gorm:"column:grade;"`  //等级 S/A/B/C/D
  GradeReason  string `json:"GradeReason" form:"GradeReason" gorm:"column:grade_reason;"`  //分级依据
  ContactReady  int `json:"ContactReady" form:"ContactReady" gorm:"column:contact_ready;"`  //1=可直接联系
  DmScript  string `json:"DmScript" form:"DmScript" gorm:"column:dm_script;"`  //私信开场话术
  Status  *int64 `json:"Status" form:"Status" gorm:"column:status;"`  //0新线索 1已联系 2有意向 3已成交 4无效
  Owner  *string `json:"Owner" form:"Owner" gorm:"column:owner;"`  //跟进人
  Remark  *string `json:"Remark" form:"Remark" gorm:"column:remark;"`  //备注
  Avatar  *string `json:"Avatar" form:"Avatar" gorm:"column:avatar;"`  //评论者头像
  VideoTitle  *string `json:"VideoTitle" form:"VideoTitle" gorm:"column:video_title;"`  //来源视频标题
  ExtractInfo  *string `json:"ExtractInfo" form:"ExtractInfo" gorm:"column:extract_info;"`  //提取信息（手机号/微信号，兜底账号ID）
  CreatedAt  *string `json:"CreatedAt" form:"CreatedAt" gorm:"column:created_at;"`
  UpdatedAt  *string `json:"UpdatedAt" form:"UpdatedAt" gorm:"column:updated_at;"`
  DeletedAt  *string `json:"DeletedAt" form:"DeletedAt" gorm:"column:deleted_at;"`
}


// TableName 获客线索 Lead自定义表名 ai_lead
func (Lead) TableName() string {
    return "ai_lead"
}






// MergeRemark 在备注后追加标记（已存在就不重复加）
func MergeRemark(old, mark string) string {
	if strings.Contains(old, mark) {
		return old
	}

	if strings.TrimSpace(old) == "" {
		return mark
	}

	return old + "｜" + mark
}
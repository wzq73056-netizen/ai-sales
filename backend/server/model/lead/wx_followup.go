package lead

import "time"

// WxFollowup 客户跟进记录（一次互动一条）
type WxFollowup struct {
	Id        int64      `json:"Id" gorm:"primaryKey;autoIncrement;column:id;"`
	NickName  string     `json:"NickName" gorm:"column:nick_name;index"`
	ChatId    string     `json:"ChatId" gorm:"column:chat_id;"`
	Kind      string     `json:"Kind" gorm:"column:kind;"`
	Content   string     `json:"Content" gorm:"column:content;"`
	MediaUrl  string     `json:"MediaUrl" gorm:"column:media_url;"`
	Operator  string     `json:"Operator" gorm:"column:operator;"`
	CreatedAt *time.Time `json:"CreatedAt" gorm:"column:created_at;"`
}

func (WxFollowup) TableName() string {
	return "ai_wx_followup"
}

// WxCustomerDetail 客户详情（合并后的客户 + 跟进记录）
type WxCustomerDetail struct {
	Customer  WxCustomer   `json:"Customer"`
	Followups []WxFollowup `json:"Followups"`
}

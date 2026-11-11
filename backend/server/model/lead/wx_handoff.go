package lead

// WxHandoff 微信转人工工单
type WxHandoff struct {
	Id         int64   `json:"Id" gorm:"primarykey;column:id;"`
	ChatId     string  `json:"ChatId" gorm:"column:chat_id;"`
	Channel    string  `json:"Channel" gorm:"column:channel;"`
	NickName   string  `json:"NickName" gorm:"column:nick_name;"`
	Category   string  `json:"Category" gorm:"column:category;"`
	Reason     string  `json:"Reason" gorm:"column:reason;"`
	LastMsg    string  `json:"LastMsg" gorm:"column:last_msg;"`
	Contact    string  `json:"Contact" gorm:"column:contact;"`
	WechatAdded int    `json:"WechatAdded" gorm:"column:wechat_added;"`
	Result     string  `json:"Result" gorm:"column:result;"`
	DraftReply string  `json:"DraftReply" gorm:"column:draft_reply;"`
	Urgency    int     `json:"Urgency" gorm:"column:urgency;"`
	Status     int     `json:"Status" gorm:"column:status;"`
	AiPaused   int     `json:"AiPaused" gorm:"column:ai_paused;"`
	ClaimUser  string  `json:"ClaimUser" gorm:"column:claim_user;"`
	ClaimAt    *string `json:"ClaimAt" gorm:"column:claim_at;"`
	CreatedAt  *string `json:"CreatedAt" gorm:"column:created_at;"`
	UpdatedAt  *string `json:"UpdatedAt" gorm:"column:updated_at;"`
}

// TableName 表名
func (WxHandoff) TableName() string {
	return "ai_wx_handoff"
}

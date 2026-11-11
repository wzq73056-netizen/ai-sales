package lead

// WxCustomer 微信客户跟进价值
type WxCustomer struct {
	Id           int64   `json:"Id" gorm:"primarykey;column:id;"`
	ChatId       string  `json:"ChatId" gorm:"column:chat_id;"`
	Side         string  `json:"Side" gorm:"column:side;"` // buyer=客户 / supplier=供货商
	NickName     string  `json:"NickName" gorm:"column:nick_name;"`
	Score        int     `json:"Score" gorm:"column:score;"`
	LlmScore     int     `json:"LlmScore" gorm:"column:llm_score;"`
	Tier         string  `json:"Tier" gorm:"column:tier;"`
	Stage        string  `json:"Stage" gorm:"column:stage;"`
	Tags         string  `json:"Tags" gorm:"column:tags;"`
	MyCount      int     `json:"MyCount" gorm:"column:my_count;"`
	TheirCount   int     `json:"TheirCount" gorm:"column:their_count;"`
	LastMsgAt    *string `json:"LastMsgAt" gorm:"column:last_msg_at;"`
	LastFromMe   int     `json:"LastFromMe" gorm:"column:last_from_me;"`
	SilentDays   int     `json:"SilentDays" gorm:"column:silent_days;"`
	FollowStatus int     `json:"FollowStatus" gorm:"column:follow_status;"`
	NextFollowAt *string `json:"NextFollowAt" gorm:"column:next_follow_at;"`
	Reason       string  `json:"Reason" gorm:"column:reason;"`
	LlmReason    string  `json:"LlmReason" gorm:"column:llm_reason;"`
	NextAction   string  `json:"NextAction" gorm:"column:next_action;"`
	CreatedAt    *string `json:"CreatedAt" gorm:"column:created_at;"`
	UpdatedAt    *string `json:"UpdatedAt" gorm:"column:updated_at;"`
}

// TableName 表名
func (WxCustomer) TableName() string {
	return "ai_wx_customer"
}

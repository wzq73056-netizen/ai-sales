package lead

// DouyinDm 抖音私信记录
type DouyinDm struct {
	Id         int64   `json:"Id" gorm:"primarykey;column:id;"`
	ProfileUrl string  `json:"ProfileUrl" gorm:"column:profile_url;"`
	Nickname   string  `json:"Nickname" gorm:"column:nickname;"`
	DouyinId   string  `json:"DouyinId" gorm:"column:douyin_id;"`
	Message    string  `json:"Message" gorm:"column:message;"`
	Status     int     `json:"Status" gorm:"column:status;"`
	Error      string  `json:"Error" gorm:"column:error;"`
	ReplyStatus int    `json:"ReplyStatus" gorm:"column:reply_status;"`
	ReplyText   string `json:"ReplyText" gorm:"column:reply_text;"`
	ReplyAt     *string `json:"ReplyAt" gorm:"column:reply_at;"`
	SentAt     *string `json:"SentAt" gorm:"column:sent_at;"`
	CreatedAt  *string `json:"CreatedAt" gorm:"column:created_at;"`
	UpdatedAt  *string `json:"UpdatedAt" gorm:"column:updated_at;"`
}

// TableName 表名
func (DouyinDm) TableName() string {
	return "ai_douyin_dm"
}

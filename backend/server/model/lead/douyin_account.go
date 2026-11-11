package lead

// DouyinAccount 抖音账号线索
type DouyinAccount struct {
	Id           int64   `json:"Id" gorm:"primarykey;column:id;"`
	Keyword      string  `json:"Keyword" gorm:"column:keyword;"`
	Nickname     string  `json:"Nickname" gorm:"column:nickname;"`
	DouyinId     string  `json:"DouyinId" gorm:"column:douyin_id;"`
	Fans         int64   `json:"Fans" gorm:"column:fans;"`
	Bio          string  `json:"Bio" gorm:"column:bio;"`
	ProfileUrl   string  `json:"ProfileUrl" gorm:"column:profile_url;"`
	IsCompetitor int     `json:"IsCompetitor" gorm:"column:is_competitor;"`
	Score        int     `json:"Score" gorm:"column:score;"`
	Status       int     `json:"Status" gorm:"column:status;"`
	Remark       string  `json:"Remark" gorm:"column:remark;"`
	FollowAt     *string `json:"FollowAt" gorm:"column:follow_at;"`
	CreatedAt    *string `json:"CreatedAt" gorm:"column:created_at;"`
	UpdatedAt    *string `json:"UpdatedAt" gorm:"column:updated_at;"`
}

// TableName 表名
func (DouyinAccount) TableName() string {
	return "ai_douyin_account"
}

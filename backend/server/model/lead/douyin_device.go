package lead

import "time"

// DouyinDevice 抖音采集账号（一个账号 = 一个独立浏览器）
type DouyinDevice struct {
	Id         uint       `json:"Id" gorm:"primaryKey;autoIncrement;column:id;"`
	Platform   string     `json:"Platform" gorm:"column:platform;" comment:"douyin/xhs"`
	Name       string     `json:"Name" gorm:"column:name;" comment:"账号备注名"`
	DouyinId   string     `json:"DouyinId" gorm:"column:douyin_id;" comment:"登录的抖音号"`
	ProfileDir string     `json:"ProfileDir" gorm:"column:profile_dir;" comment:"浏览器用户数据目录"`
	BrowserPath string    `json:"BrowserPath" gorm:"column:browser_path;" comment:"浏览器程序路径"`
	BrowserName string    `json:"BrowserName" gorm:"column:browser_name;" comment:"浏览器名称"`
	CdpPort    int        `json:"CdpPort" gorm:"column:cdp_port;" comment:"CDP 调试端口"`
	Status     int        `json:"Status" gorm:"column:status;" comment:"1启用 0停用"`
	LastSeen   *time.Time `json:"LastSeen" gorm:"column:last_seen;"`
	CreatedAt  *time.Time `json:"CreatedAt" gorm:"column:created_at;"`
	UpdatedAt  *time.Time `json:"UpdatedAt" gorm:"column:updated_at;"`
}

func (DouyinDevice) TableName() string {
	return "ai_douyin_device"
}

// DouyinWhitelist 私信自动回复白名单
type DouyinWhitelist struct {
	Id          uint       `json:"Id" gorm:"primaryKey;autoIncrement;column:id;"`
	Platform    string     `json:"Platform" gorm:"column:platform;"`
	DeviceId    uint       `json:"DeviceId" gorm:"column:device_id;index"`
	Nickname    string     `json:"Nickname" gorm:"column:nickname;"`
	DouyinId    string     `json:"DouyinId" gorm:"column:douyin_id;"`
	ProfileUrl  string     `json:"ProfileUrl" gorm:"column:profile_url;"`
	Enabled     int        `json:"Enabled" gorm:"column:enabled;"`
	LastReplyAt *time.Time `json:"LastReplyAt" gorm:"column:last_reply_at;"`
	CreatedAt   *time.Time `json:"CreatedAt" gorm:"column:created_at;"`
	UpdatedAt   *time.Time `json:"UpdatedAt" gorm:"column:updated_at;"`
}

func (DouyinWhitelist) TableName() string {
	return "ai_douyin_whitelist"
}

// XhsSchedule 小红书定时采集配置
type XhsSchedule struct {
	Id           uint   `json:"Id" gorm:"column:id;primaryKey"`
	Keywords     string `json:"Keywords" gorm:"column:keywords"`
	Notes        int    `json:"Notes" gorm:"column:notes"`
	CommentLimit int    `json:"CommentLimit" gorm:"column:comment_limit"`
	RunAt        string `json:"RunAt" gorm:"column:run_at"`
	Enabled      int    `json:"Enabled" gorm:"column:enabled"`
	LastRunDate  string `json:"LastRunDate" gorm:"column:last_run_date"`
}
// DmMessageInput 监听器上报的一条私信
type DmMessageInput struct {
	PeerId    string `json:"peerId"`
	PeerName  string `json:"peerName"`
	Text      string `json:"text"`
	Direction int    `json:"direction"`
	Time      string `json:"time"`
}

// ScrapeKeyword 采集关键词
type ScrapeKeyword struct {
	Id        uint   `json:"Id"`
	Platform  string `json:"Platform"`
	Keyword   string `json:"Keyword"`
	Enabled   int    `json:"Enabled" gorm:"column:enabled"`
	LastRunAt string `json:"LastRunAt"`
}

// ScrapeBlacklist 采集黑名单
type ScrapeBlacklist struct {
	Id     uint   `json:"Id"`
	Kind   string `json:"Kind"`
	Value  string `json:"Value"`
	Reason string `json:"Reason"`
}
// WxAddConfig 微信加好友配置
type WxAddConfig struct {
	Id           uint   `json:"Id"`
	Enabled      int    `json:"Enabled"`
	DailyLimit   int    `json:"DailyLimit"`
	IntervalMin  int    `json:"IntervalMin"`
	IntervalMax  int    `json:"IntervalMax"`
	PauseMinutes int    `json:"PauseMinutes"`
	WindowStart  string `json:"WindowStart"`
	WindowEnd    string `json:"WindowEnd"`
	Remark       string `json:"Remark"`
	PausedUntil  string `json:"PausedUntil"`
	LastRunAt    string `json:"LastRunAt"`
}
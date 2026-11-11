package lead

// DouyinTask 抖音获客任务
type DouyinTask struct {
	Id               int64   `json:"Id" gorm:"primarykey;column:id;"`
	Industry         string  `json:"Industry" gorm:"column:industry;"`
	Name             string  `json:"Name" gorm:"column:name;"`
	DeviceId         uint    `json:"DeviceId" gorm:"column:device_id;"`
	CrawlType        int     `json:"CrawlType" gorm:"column:crawl_type;"` // 0视频获客 1账号获客
	Keywords         string  `json:"Keywords" gorm:"column:keywords;"`
	VideosPerKeyword int     `json:"VideosPerKeyword" gorm:"column:videos_per_keyword;"`
	MaxDays          int     `json:"MaxDays" gorm:"column:max_days;"`
	LimitN           int     `json:"LimitN" gorm:"column:limit_n;"`
	City             string  `json:"City" gorm:"column:city;"`
	ExcludeWords     string  `json:"ExcludeWords" gorm:"column:exclude_words;"`
	Status           int     `json:"Status" gorm:"column:status;"`
	ScheduleType     string  `json:"ScheduleType" gorm:"column:schedule_type;"`
	Schedule         string  `json:"Schedule" gorm:"column:schedule;"`
	NextRunAt        *string `json:"NextRunAt" gorm:"column:next_run_at;"`
	StartedAt        *string `json:"StartedAt" gorm:"column:started_at;"`
	FinishedAt       *string `json:"FinishedAt" gorm:"column:finished_at;"`
	VideosCount      int     `json:"VideosCount" gorm:"column:videos_count;"`
	CommentsCount    int     `json:"CommentsCount" gorm:"column:comments_count;"`
	LeadsCount       int     `json:"LeadsCount" gorm:"column:leads_count;"`
	Log              string  `json:"Log" gorm:"column:log;"`
	CreatedAt        string  `json:"CreatedAt" gorm:"column:created_at;"`
	UpdatedAt        string  `json:"UpdatedAt" gorm:"column:updated_at;"`
}

// TableName 表名
func (DouyinTask) TableName() string {
	return "ai_douyin_task"
}

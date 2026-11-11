package lead

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

var schedOnce sync.Once

// ensureScheduler 启动定时扫描（每 60 秒）
func ensureScheduler() {
	schedOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if RunningTaskId() != 0 {
					continue
				}

				var task leadModel.DouyinTask
				err := global.GVA_DB.
					Where("status = 0 AND next_run_at IS NOT NULL AND next_run_at <= ?", time.Now().Format("2006-01-02 15:04:05")).
					Order("next_run_at ASC").First(&task).Error
				if err != nil {
					continue
				}

				_ = runTaskAsync(&task)
			}
		}()
	})
}

// CreateDouyinTask 创建任务
func (leadService *LeadService) CreateDouyinTask(t *leadModel.DouyinTask, autoStart bool) (leadModel.DouyinTask, error) {
	if strings.TrimSpace(t.Keywords) == "" {
		return *t, fmt.Errorf("关键词不能为空")
	}
	if t.VideosPerKeyword <= 0 {
		t.VideosPerKeyword = 3
	}
	if t.MaxDays <= 0 {
		t.MaxDays = 180
	}
	if t.LimitN <= 0 {
		t.LimitN = 10
	}
	if t.Name == "" {
		first := strings.Split(t.Keywords, ",")[0]
		t.Name = strings.TrimSpace(t.Industry + " · " + first)
	}
	if t.ScheduleType == "" {
		t.ScheduleType = "manual"
	}

	t.Status = 0
	t.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	t.UpdatedAt = t.CreatedAt

	if t.ScheduleType != "manual" {
		if next := nextRunTime(t, time.Now()); next != "" {
			t.NextRunAt = &next
		}
	}

	if err := global.GVA_DB.Create(t).Error; err != nil {
		return *t, err
	}

	ensureScheduler()

	if autoStart && t.ScheduleType == "manual" {
		_ = runTaskAsync(t)
	}

	return *t, nil
}

// GetDouyinTaskList 任务列表
func (leadService *LeadService) GetDouyinTaskList(page, pageSize int) ([]leadModel.DouyinTask, int64, error) {
	var list []leadModel.DouyinTask
	var total int64

	db := global.GVA_DB.Model(&leadModel.DouyinTask{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	err := db.Order("id DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&list).Error
	return list, total, err
}

// StartDouyinTask 手动启动任务
func (leadService *LeadService) StartDouyinTask(id int64) error {
	var task leadModel.DouyinTask
	if err := global.GVA_DB.First(&task, id).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}
	return runTaskAsync(&task)
}

// DeleteDouyinTask 删除任务
func (leadService *LeadService) DeleteDouyinTask(id int64) error {
	return global.GVA_DB.Delete(&leadModel.DouyinTask{}, "id = ?", id).Error
}

// GenKeywords 依据行业用大模型生成搜索关键词
func (leadService *LeadService) GenKeywords(industry string, typ int) ([]string, error) {
	llm := loadLLMConfig()
	baseURL, apiKey := llm.BaseURL, llm.ApiKey
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("未配置大模型，无法生成关键词")
	}
	model := llm.Model
	if model == "" {
		model = "deepseek-chat"
	}

	taskDesc := "视频搜索词（用户会拿它搜视频，进而从评论区找线索）"
	if typ == 3 {
		taskDesc = "账号搜索词（用户会拿它搜抖音用户，目标是找到潜在客户或对标账号的人群）"
	}

	prompt := fmt.Sprintf(
		"你是抖音获客关键词专家。行业：%s。\n"+
			"本次需要：%s。\n"+
			"请生成 6 个中文关键词，要求：\n"+
			"1) 是真实用户会主动搜索的说法（口语化、具体），不是广告词；\n"+
			"2) 覆盖价格、场景、身份三类；\n"+
			"3) 每个不超过 10 个字。\n"+
			"只输出 JSON 数组，例如 [\"代理记账\",\"小规模纳税人\"]，不要任何解释。",
		industry, taskDesc,
	)

	payload, _ := json.Marshal(map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.7,
	})

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("模型调用失败(%d)", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("模型返回解析失败")
	}

	raw := strings.TrimSpace(parsed.Choices[0].Message.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")

	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("模型返回格式异常")
	}

	var keywords []string
	if err := json.Unmarshal([]byte(raw[start:end+1]), &keywords); err != nil {
		return nil, err
	}

	return keywords, nil
}

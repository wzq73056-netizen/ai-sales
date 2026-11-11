package lead

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 小红书采集：手动触发 + 状态查询
var (
	xhsMu       sync.Mutex
	xhsRunning  bool
	xhsStarted  string
	xhsFinished string
	xhsLogPath  = `E:\ai\data\logs\xhs_collect.log`
)

// XhsCollectStatus 采集状态
type XhsCollectStatus struct {
	Running  bool   `json:"running"`
	Started  string `json:"started"`
	Finished string `json:"finished"`
	Log      string `json:"log"`
}

// StartXhsCollect 触发一次小红书采集
func (leadService *LeadService) StartXhsCollect(keywords string, notes int, limit int) (XhsCollectStatus, error) {
	xhsMu.Lock()
	defer xhsMu.Unlock()

	if xhsRunning {
		return XhsCollectStatus{Running: true, Started: xhsStarted}, fmt.Errorf("已有采集在跑")
	}

	// 风控暂停中就别再触发
	if paused, minutes := throttlePaused(); paused {
		return XhsCollectStatus{}, fmt.Errorf("风控暂停中，还需 %d 分钟，稍后再试", minutes)
	}

	if keywords == "" {
		keywords = "鸡蛋批发"
	}

	if notes <= 0 || notes > 10 {
		notes = 3
	}

	if limit <= 0 || limit > 100 {
		limit = 30
	}

	if err := os.MkdirAll(filepath.Dir(xhsLogPath), 0o755); err != nil {
		return XhsCollectStatus{}, err
	}

	logFile, err := os.Create(xhsLogPath)
	if err != nil {
		return XhsCollectStatus{}, err
	}

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", `E:\ai\scripts\xhs_batch.ps1`,
		"-KeywordsCsv", keywords,
		"-NotesPerKeyword", fmt.Sprintf("%d", notes),
		"-Limit", fmt.Sprintf("%d", limit),
		"-CdpPort", "9333")

	cmd.Dir = `E:\ai\scraper`
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return XhsCollectStatus{}, err
	}

	xhsRunning = true
	xhsStarted = time.Now().Format("2006-01-02 15:04:05")
	xhsFinished = ""

	// 写一条采集历史
	run := map[string]any{
		"keywords":      keywords,
		"notes":         notes,
		"comment_limit": limit,
		"status":        0,
		"started_at":    time.Now(),
	}
	global.GVA_DB.Table("ai_xhs_run").Create(run)

	var runId uint
	global.GVA_DB.Table("ai_xhs_run").Select("MAX(id)").Scan(&runId)

	go func() {
		_ = cmd.Wait()
		logFile.Close()

		xhsMu.Lock()
		xhsRunning = false
		xhsFinished = time.Now().Format("2006-01-02 15:04:05")
		xhsMu.Unlock()

		// 采完自动推送线索
		exec.Command("node", `E:\ai\scraper\xhs_push.mjs`, `E:\ai\data`).Run()

		// 收尾：解析 STATS 写回历史
		if runId > 0 {
			notesDone := 0
			comments := 0
			status := 1
			message := "完成"

			if data, err := os.ReadFile(xhsLogPath); err == nil {
				text := string(data)

				if idx := strings.Index(text, "STATS "); idx >= 0 {
					line := strings.SplitN(text[idx+6:], "\n", 2)[0]

					var stats struct {
						Videos     int `json:"videos"`
						Comments   int `json:"comments"`
						SkippedDup int `json:"skippedDup"`
					}

					if json.Unmarshal([]byte(strings.TrimSpace(line)), &stats) == nil {
						notesDone = stats.Videos
						comments = stats.Comments
						message = fmt.Sprintf("完成，去重跳过 %d 篇", stats.SkippedDup)
					}
				}

				if strings.Contains(text, "搜索失败") || strings.Contains(text, "BLOCKED") {
					status = 2
					message = "部分失败，看日志"
				}

				tail := text
				if len(tail) > 1500 {
					tail = tail[len(tail)-1500:]
				}

				global.GVA_DB.Table("ai_xhs_run").Where("id = ?", runId).Updates(map[string]any{
					"notes_done":    notesDone,
					"comments_kept": comments,
					"status":        status,
					"message":       message,
					"log_tail":      tail,
					"finished_at":   time.Now(),
				})
			}
		}
	}()

	return XhsCollectStatus{Running: true, Started: xhsStarted}, nil
}

// GetXhsCollectStatus 采集状态 + 日志尾部
func (leadService *LeadService) GetXhsCollectStatus() XhsCollectStatus {
	xhsMu.Lock()
	status := XhsCollectStatus{Running: xhsRunning, Started: xhsStarted, Finished: xhsFinished}
	xhsMu.Unlock()

	if data, err := os.ReadFile(xhsLogPath); err == nil {
		lines := strings.Split(string(data), "\n")
		if len(lines) > 40 {
			lines = lines[len(lines)-40:]
		}

		status.Log = strings.Join(lines, "\n")
	}

	return status
}

// XhsLeadStats 小红书线索统计
func (leadService *LeadService) XhsLeadStats() map[string]any {
	result := map[string]any{}

	var total int64
	global.GVA_DB.Model(&struct{}{}).Table("ai_lead").Where("source = ?", "xhs").Count(&total)
	result["total"] = total

	type row struct {
		Lv    string `json:"lv"`
		Count int    `json:"count"`
	}

	var rows []row

	global.GVA_DB.Table("ai_lead").
		Select("CONCAT(grade, IF(contact_ready = 1, '1', '2')) AS lv, COUNT(*) AS count").
		Where("source = ?", "xhs").
		Group("lv").
		Scan(&rows)

	result["grades"] = rows

	var todayCount int64
	global.GVA_DB.Table("ai_lead").
		Where("source = ? AND DATE(created_at) = ?", "xhs", time.Now().Format("2006-01-02")).
		Count(&todayCount)
	result["today"] = todayCount

	return result
}

// XhsRun 采集记录
type XhsRun struct {
	Id           uint   `json:"Id"`
	Keywords     string `json:"Keywords"`
	Notes        int    `json:"Notes"`
	NotesDone    int    `json:"NotesDone"`
	CommentsKept int    `json:"CommentsKept"`
	Status       int    `json:"Status"`
	Message      string `json:"Message"`
	LogTail      string `json:"LogTail"`
	SkippedDup   int    `json:"SkippedDup"`
	StartedAt    string `json:"StartedAt"`
	FinishedAt   string `json:"FinishedAt"`
}

// GetXhsRuns 采集历史
func (leadService *LeadService) GetXhsRuns(limit int) []XhsRun {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var runs []XhsRun

	global.GVA_DB.Table("ai_xhs_run").
		Select("id, keywords, notes, notes_done, comments_kept, status, message, log_tail, started_at, finished_at").
		Order("id desc").Limit(limit).Scan(&runs)

	return runs
}
// SendXhsDm 主动私信（只允许白名单）
func (leadService *LeadService) SendXhsDm(nick, message string, aiHint string, send bool) (map[string]any, error) {
	if nick == "" {
		return nil, fmt.Errorf("缺少昵称")
	}

	// 后端再校验一次白名单，避免脚本被绕过
	var count int64
	global.GVA_DB.Table("ai_douyin_whitelist").
		Where("platform = ? AND nickname = ? AND enabled = 1", "xhs", nick).Count(&count)

	if count == 0 {
		return nil, fmt.Errorf("%s 不在小红书私信白名单", nick)
	}

	args := []string{`E:\ai\scraper\xhs_dm_send.mjs`, "--nick", nick}

	if message != "" {
		args = append(args, "--message", message)
	}

	if aiHint != "" {
		args = append(args, "--ai", aiHint)
	}

	if send {
		args = append(args, "--send")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", args...)
	cmd.Dir = `E:\ai\scraper`

	// Playwright 需要可写的临时目录
	_ = os.MkdirAll(`E:\ai\data\tmp`, 0o755)
	cmd.Env = append(os.Environ(),
		`TEMP=E:\ai\data\tmp`,
		`TMP=E:\ai\data\tmp`,
	)

	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))

	if err != nil {
		return map[string]any{"ok": false, "error": err.Error(), "output": text}, nil
	}

	return map[string]any{"ok": true, "output": text}, nil
}
// SendDyDm 抖音主动私信（只允许白名单）
func (leadService *LeadService) SendDyDm(nick, message, aiHint string, send bool) (map[string]any, error) {
	if nick == "" {
		return nil, fmt.Errorf("缺少昵称")
	}

	// 白名单不再是门槛：有登记就用它做抖音号校验，没有也能发
	var white struct {
		DouyinId string
	}
	global.GVA_DB.Table("ai_douyin_whitelist").
		Select("douyin_id").Where("platform = ? AND nickname = ?", "douyin", nick).
		Scan(&white)

	args := []string{`E:\ai\scraper\dy_dm_proactive.mjs`, "--nick", nick}

	if white.DouyinId != "" {
		args = append(args, "--douyin-id", white.DouyinId)
	}

	if message != "" {
		args = append(args, "--message", message)
	}

	if aiHint != "" {
		args = append(args, "--ai", aiHint)
	}

	if send {
		args = append(args, "--send")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", args...)
	cmd.Dir = `E:\ai\scraper`
	_ = os.MkdirAll(`E:\ai\data\tmp`, 0o755)
	cmd.Env = append(os.Environ(), `TEMP=E:\ai\data\tmp`, `TMP=E:\ai\data\tmp`)

	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))

	if err != nil {
		return map[string]any{"ok": false, "error": err.Error(), "output": text}, nil
	}

	if send && strings.Contains(text, `"sent":true`) {
		leadService.RecordCampaign(0, "douyin", nick, nick, message)
	}

	return map[string]any{"ok": true, "output": text}, nil
}

// SendXhsProactiveDm 给线索主动私信（默认只预览）
func (leadService *LeadService) SendXhsProactiveDm(leadId uint, message, aiHint string, send bool) (map[string]any, error) {
	var lead struct {
		ContactName string
		SourceUrl   string
	}

	if err := global.GVA_DB.Table("ai_lead").
		Select("contact_name, source_url").Where("id = ?", leadId).Scan(&lead).Error; err != nil {
		return nil, fmt.Errorf("线索不存在")
	}

	if lead.SourceUrl == "" {
		return nil, fmt.Errorf("该线索没有主页链接（缺 xsec_token），无法主动私信")
	}

	profile := strings.TrimPrefix(lead.SourceUrl, "https://www.xiaohongshu.com")

	args := []string{`E:\ai\scraper\xhs_dm_proactive.mjs`,
		"--nick", lead.ContactName, "--profile", profile}

	if message != "" {
		args = append(args, "--message", message)
	}

	if aiHint != "" {
		args = append(args, "--ai", aiHint)
	}

	// 默认只预览，必须显式传 send=true 才真发
	if send {
		args = append(args, "--send")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", args...)
	cmd.Dir = `E:\ai\scraper`
	_ = os.MkdirAll(`E:\ai\data\tmp`, 0o755)
	cmd.Env = append(os.Environ(), `TEMP=E:\ai\data\tmp`, `TMP=E:\ai\data\tmp`)

	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))

	if err != nil {
		return map[string]any{"ok": false, "error": err.Error(), "output": text}, nil
	}

	// 真发成功 → 记一条，用于统计回复率
	if send && strings.Contains(text, `"sent":true`) {
		leadService.RecordCampaign(leadId, "xhs", lead.ContactName, lead.ContactName, message)
	}

	return map[string]any{"ok": true, "output": text}, nil
}
// throttlePaused 读取节流状态（被风控时脚本会暂停一段时间）
func throttlePaused() (bool, int) {
	const file = `E:\ai\data\scrape_throttle.json`

	data, err := os.ReadFile(file)
	if err != nil {
		return false, 0
	}

	var state struct {
		Level       int    `json:"level"`
		PausedUntil string `json:"pausedUntil"`
	}

	if json.Unmarshal(data, &state) != nil || state.PausedUntil == "" {
		return false, 0
	}

	until, err := time.ParseInLocation("2006-01-02 15:04:05", state.PausedUntil, time.Local)
	if err != nil {
		return false, 0
	}

	left := int(time.Until(until).Minutes())
	if left > 0 {
		return true, left
	}

	return false, 0
}
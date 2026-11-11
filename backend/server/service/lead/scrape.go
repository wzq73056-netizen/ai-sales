package lead

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

// TaskStats 采集脚本结束时输出的统计
type TaskStats struct {
	Videos   int `json:"videos"`
	Comments int `json:"comments"`
	Leads    int `json:"leads"`
}

var (
	runMu         sync.Mutex
	runningTaskId int64
	runningCmd    *exec.Cmd
	runningLog    []string
)

// RunningTaskId 当前运行中的任务 ID（0 表示空闲）
func RunningTaskId() int64 {
	runMu.Lock()
	defer runMu.Unlock()
	return runningTaskId
}

// RunningLog 当前运行日志
func RunningLog() []string {
	runMu.Lock()
	defer runMu.Unlock()
	return append([]string{}, runningLog...)
}

// StopCurrent 停止当前任务
func StopCurrent() error {
	runMu.Lock()
	defer runMu.Unlock()

	if runningTaskId == 0 || runningCmd == nil || runningCmd.Process == nil {
		return fmt.Errorf("当前没有运行中的任务")
	}

	if err := runningCmd.Process.Kill(); err != nil {
		return err
	}
	runningLog = append(runningLog, "已手动停止")
	return nil
}

func appendRunLog(line string) {
	runMu.Lock()
	defer runMu.Unlock()

	runningLog = append(runningLog, line)
	if len(runningLog) > 300 {
		runningLog = runningLog[len(runningLog)-300:]
	}
}

// runTaskAsync 后台执行采集任务并回写数据库
func runTaskAsync(t *leadModel.DouyinTask) error {
	runMu.Lock()
	if runningTaskId != 0 {
		runMu.Unlock()
		return fmt.Errorf("已有任务在运行（ID=%d）", runningTaskId)
	}
	runningTaskId = t.Id
	runningLog = []string{fmt.Sprintf("开始采集：%s", t.Keywords)}
	runMu.Unlock()

	now := time.Now().Format("2006-01-02 15:04:05")
	global.GVA_DB.Model(&leadModel.DouyinTask{}).Where("id = ?", t.Id).Updates(map[string]any{
		"status":         1,
		"started_at":     now,
		"log":            "",
		"videos_count":   0,
		"comments_count": 0,
		"leads_count":    0,
	})

	args := []string{
		"-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", `E:\ai\scripts\dy_batch.ps1`,
		"-KeywordsCsv", t.Keywords,
		"-VideosPerKeyword", fmt.Sprintf("%d", t.VideosPerKeyword),
		"-MaxDays", fmt.Sprintf("%d", t.MaxDays),
		"-Limit", fmt.Sprintf("%d", t.LimitN),
	}
	if t.City != "" {
		args = append(args, "-City", t.City)
	}
	if t.ExcludeWords != "" {
		args = append(args, "-Exclude", t.ExcludeWords)
	}
	if t.CrawlType == 1 {
		args = append(args, "-CrawlType", "account")
	}

	// 指定账号对应的浏览器端口
	cdpPort := 9333

	if t.DeviceId > 0 {
		var device leadModel.DouyinDevice
		if err := global.GVA_DB.First(&device, t.DeviceId).Error; err == nil && device.CdpPort > 0 {
			cdpPort = device.CdpPort
		}
	}

	args = append(args, "-CdpPort", fmt.Sprintf("%d", cdpPort))

	cmd := exec.Command("powershell", args...)
	cmd.Dir = `E:\ai\scraper`

	// 不要把 stdout 接到管道：Node 在管道下会 EPERM，改成写日志文件再跟读
	logPath := fmt.Sprintf(`E:\ai\data\logs\scrape-%d.log`, t.Id)
	if err := os.MkdirAll(`E:\ai\data\logs`, 0o755); err != nil {
		return err
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return err
	}

	runMu.Lock()
	runningCmd = cmd
	runMu.Unlock()

	taskId := t.Id

	go func() {
		stats := TaskStats{}
		stopped := false
		blocked := ""
		offset := int64(0)
		partial := ""

		// 每秒跟读一次日志文件
		ticker := time.NewTicker(time.Second)
		go func() {
			for range ticker.C {
				fh, err := os.Open(logPath)
				if err != nil {
					continue
				}

				info, err := fh.Stat()
				if err == nil && info.Size() > offset {
					buf := make([]byte, info.Size()-offset)
					if _, err := fh.ReadAt(buf, offset); err == nil {
						offset = info.Size()

						text := partial + string(buf)
						lines := strings.Split(text, "\n")
						partial = lines[len(lines)-1]

						for _, raw := range lines[:len(lines)-1] {
							line := strings.TrimSpace(raw)
							if line == "" {
								continue
							}

							if strings.HasPrefix(line, "STATS ") {
								_ = json.Unmarshal([]byte(strings.TrimPrefix(line, "STATS ")), &stats)
							}

							if strings.HasPrefix(line, "BLOCKED ") {
								blocked = strings.TrimSpace(strings.TrimPrefix(line, "BLOCKED "))
							}

							if strings.Contains(line, "已手动停止") {
								stopped = true
							}

							appendRunLog(line)
						}
					}
				}

				fh.Close()
			}
		}()

		waitErr := cmd.Wait()
		ticker.Stop()

		// 进程退出后再补读一次，确保 STATS 最后一行不丢
		if final, err := os.ReadFile(logPath); err == nil {
			for _, raw := range strings.Split(string(final), "\n") {
				line := strings.TrimSpace(raw)

				if strings.HasPrefix(line, "STATS ") {
					_ = json.Unmarshal([]byte(strings.TrimPrefix(line, "STATS ")), &stats)
				}

				if strings.HasPrefix(line, "BLOCKED ") {
					blocked = strings.TrimSpace(strings.TrimPrefix(line, "BLOCKED "))
				}

				if strings.Contains(line, "已手动停止") {
					stopped = true
				}
			}
		}

		runMu.Lock()
		finalStatus := 2
		if waitErr != nil {
			finalStatus = 3
		}
		if stopped {
			finalStatus = 4
		}
		if blocked != "" {
			finalStatus = 3
			runningLog = append(runningLog, "任务失败：触发抖音风控/验证（"+blocked+"），本轮数据不可信")
		}
		logText := strings.Join(runningLog, "\n")
		runningTaskId = 0
		runningCmd = nil
		runMu.Unlock()

		finish := time.Now().Format("2006-01-02 15:04:05")
		updates := map[string]any{
			"status":         finalStatus,
			"finished_at":    finish,
			"log":            logText,
			"videos_count":   stats.Videos,
			"comments_count": stats.Comments,
			"leads_count":    stats.Leads,
		}

		var task leadModel.DouyinTask
		if err := global.GVA_DB.First(&task, taskId).Error; err == nil {
			if next := nextRunTime(&task, time.Now()); next != "" {
				updates["next_run_at"] = next
				updates["status"] = 0
			}
		}

		global.GVA_DB.Model(&leadModel.DouyinTask{}).Where("id = ?", taskId).Updates(updates)
	}()

	return nil
}

// nextRunTime 依据调度配置计算下次执行时间
func nextRunTime(t *leadModel.DouyinTask, now time.Time) string {
	switch t.ScheduleType {
	case "daily":
		var hh, mm int
		if _, err := fmt.Sscanf(t.Schedule, "%d:%d", &hh, &mm); err != nil {
			return ""
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		return next.Format("2006-01-02 15:04:05")
	case "interval":
		var minutes int
		if _, err := fmt.Sscanf(t.Schedule, "%d", &minutes); err != nil || minutes <= 0 {
			return ""
		}
		return now.Add(time.Duration(minutes) * time.Minute).Format("2006-01-02 15:04:05")
	}
	return ""
}

// StopCurrent 供 API 层调用
func (leadService *LeadService) StopCurrent() error {
	return StopCurrent()
}

// RunningInfo 供 API 层调用
func (leadService *LeadService) RunningInfo() (int64, []string) {
	return RunningTaskId(), RunningLog()
}

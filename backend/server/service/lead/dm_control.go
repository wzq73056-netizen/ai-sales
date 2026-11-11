package lead

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const (
	dmControlFile = `E:\ai\data\dm_control.json`
	dmStatusFile  = `E:\ai\data\dm_watchd_status.json`
	dmDaemonCmd   = `E:\ai\scraper\dy_dm_watchd.mjs`
)

// DmControl 私信自动化开关
type DmControl struct {
	Listen    bool `json:"listen"`    // 是否监听会话列表
	AutoReply bool `json:"autoReply"` // 是否 AI 生成回复
	AutoSend  bool `json:"autoSend"`  // 是否真实发送（false = 只预演）
	Interval  int  `json:"interval"`  // 轮询秒数
}

// DmStatus 守护进程状态
type DmStatus struct {
	Running      bool   `json:"Running"`
	Listening    bool   `json:"Listening"`
	AutoReply    bool   `json:"AutoReply"`
	AutoSend     bool   `json:"AutoSend"`
	Rounds       int    `json:"Rounds"`
	Conversations int   `json:"Conversations"`
	Changes      int    `json:"Changes"`
	LastCheckAt  string `json:"LastCheckAt"`
	LastError    string `json:"LastError"`
	LastAutoReply string `json:"LastAutoReply"`
}

var (
	dmMu      sync.Mutex
	dmCmd     *exec.Cmd
	dmRunning bool
)

// GetDmControl 读取开关配置
func (leadService *LeadService) GetDmControl() DmControl {
	ctl := DmControl{Interval: 60}

	raw, err := os.ReadFile(dmControlFile)
	if err != nil {
		return ctl
	}

	_ = json.Unmarshal(raw, &ctl)
	if ctl.Interval <= 0 {
		ctl.Interval = 60
	}

	return ctl
}

// SetDmControl 保存开关；开启监听时确保守护进程在跑
func (leadService *LeadService) SetDmControl(ctl DmControl) error {
	if ctl.Interval <= 0 {
		ctl.Interval = 60
	}

	data, _ := json.MarshalIndent(ctl, "", "  ")

	if err := os.MkdirAll(filepath.Dir(dmControlFile), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dmControlFile, data, 0o644); err != nil {
		return err
	}

	if ctl.Listen {
		return StartDmDaemon()
	}

	StopDmDaemon()
	return nil
}

// killStaleDaemons 清掉命令行里带 dy_dm_watchd 的残留 node 进程
func killStaleDaemons() {
	ps := exec.Command("powershell", "-NoProfile", "-Command",
		`Get-CimInstance Win32_Process -Filter "Name='node.exe'" | Where-Object { $_.CommandLine -like '*dy_dm_watchd*' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }`)
	ps.Dir = `E:\ai`
	_ = ps.Run()
}

// StartDmDaemon 启动后台守护进程（幂等）
func StartDmDaemon() error {
	dmMu.Lock()
	defer dmMu.Unlock()

	if dmCmd != nil && dmCmd.Process != nil && dmRunning {
		return nil
	}

	killStaleDaemons()

	cmd := exec.Command("node", dmDaemonCmd)
	cmd.Dir = `E:\ai\scraper`

	// Playwright 默认用系统 TEMP，后台进程下可能没有权限，改到工作区
	tmpDir := `E:\ai\data\tmp`
	_ = os.MkdirAll(tmpDir, 0o755)
	cmd.Env = append(os.Environ(), "TEMP="+tmpDir, "TMP="+tmpDir, "TMPDIR="+tmpDir)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动守护进程失败: %w", err)
	}

	dmCmd = cmd
	dmRunning = true

	go func(c *exec.Cmd) {
		_ = c.Wait()

		dmMu.Lock()
		if dmCmd == c {
			dmRunning = false
			dmCmd = nil
		}
		dmMu.Unlock()
	}(cmd)

	return nil
}

// StopDmDaemon 停止守护进程
func StopDmDaemon() {
	dmMu.Lock()
	defer dmMu.Unlock()

	if dmCmd != nil && dmCmd.Process != nil {
		_ = dmCmd.Process.Kill()
	}

	dmRunning = false
	dmCmd = nil
}

// DmDaemonRunning 守护进程是否在跑
func DmDaemonRunning() bool {
	dmMu.Lock()
	defer dmMu.Unlock()

	return dmRunning
}

// GetDmStatus 状态（合并守护进程写出的状态文件）
func (leadService *LeadService) GetDmStatus() DmStatus {
	ctl := leadService.GetDmControl()

	st := DmStatus{
		Running:   DmDaemonRunning(),
		Listening: ctl.Listen,
		AutoReply: ctl.AutoReply,
		AutoSend:  ctl.AutoSend,
	}

	raw, err := os.ReadFile(dmStatusFile)
	if err != nil {
		return st
	}

	var file struct {
		Rounds        int    `json:"rounds"`
		Conversations int    `json:"conversations"`
		Changes       int    `json:"changes"`
		LastCheckAt   string `json:"lastCheckAt"`
		LastError     string `json:"lastError"`
		LastAutoReply string `json:"lastAutoReply"`
	}

	if err := json.Unmarshal(raw, &file); err == nil {
		st.Rounds = file.Rounds
		st.Conversations = file.Conversations
		st.Changes = file.Changes
		st.LastCheckAt = file.LastCheckAt
		st.LastError = file.LastError
		st.LastAutoReply = file.LastAutoReply
	}

	return st
}

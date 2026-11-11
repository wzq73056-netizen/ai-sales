package lead

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Daemon 监听进程（抖音/小红书/微信）
//
// 这些进程是常驻的采集/监听脚本，之前只能在电脑上用 start-all.ps1 起，
// 现在通过接口由后台托管：记录 PID、状态、日志位置。
type Daemon struct {
	Key       string `json:"Key"`
	Name      string `json:"Name"`
	Desc      string `json:"Desc"`
	Alive     bool   `json:"Alive"`
	Pid       int    `json:"Pid"`
	StartedAt string `json:"StartedAt"`
	Heartbeat string `json:"Heartbeat"`
	LogFile   string `json:"LogFile"`
	// 日志最后一行，进程挂掉时用来看原因（比如微信掉登录）
	LastLog string `json:"LastLog"`
}

type daemonSpec struct {
	key     string
	name    string
	desc    string
	program string
	args    []string
	workdir string
	logName string
	// 心跳文件，存在且较新说明脚本在跑
	heartbeat string
}

type daemonState struct {
	Pid       int    `json:"pid"`
	StartedAt string `json:"started_at"`
}

const (
	daemonStateFile = `E:\ai\data\daemons.json`
	daemonLogDir    = `E:\ai\data\logs`
	nodeExe         = `C:\Users\11\.dsh\dsh-runtimes\dsh-primary-runtime\dependencies\node\bin\node.exe`
	pythonExe       = `C:\Users\11\.dsh\dsh-runtimes\dsh-primary-runtime\dependencies\python\python.exe`
	scraperDir      = `E:\ai\scraper`
	scriptsDir      = `E:\ai\scripts`
)

var daemonMu sync.Mutex

func daemonSpecs() map[string]daemonSpec {
	return map[string]daemonSpec{
		"douyin": {
			key:       "douyin",
			name:      "抖音监听",
			desc:      "评论区私信监听，自动回复；只读采集，不发消息",
			program:   nodeExe,
			args:      []string{filepath.Join(scraperDir, "dy_dm_watch.mjs")},
			workdir:   scraperDir,
			logName:   "dywatch",
			heartbeat: `E:\ai\data\dy_heartbeat-1.txt`,
		},
		"xhs": {
			key:       "xhs",
			name:      "小红书监听",
			desc:      "私信会话监听，把三个平台的会话汇总到收件箱",
			program:   nodeExe,
			args:      []string{filepath.Join(scraperDir, "xhs_dm_watch.mjs")},
			workdir:   scraperDir,
			logName:   "xhswatch",
			heartbeat: `E:\ai\data\xhs_heartbeat.txt`,
		},
		"wechat": {
			key:       "wechat",
			name:      "微信监听",
			desc:      "读本地微信消息、自动回复、执行加好友任务",
			program:   pythonExe,
			args:      []string{filepath.Join(scriptsDir, "wx_watch.py"), "--interval", "1.5", "--reply", "--send"},
			workdir:   scriptsDir,
			logName:   "wxwatch",
			heartbeat: `E:\ai\data\wx_heartbeat.txt`,
		},
	}
}

// GetDaemons 返回三个监听进程的当前状态
func (leadService *LeadService) GetDaemons() ([]Daemon, error) {
	specs := daemonSpecs()
	states := loadDaemonState()

	order := []string{"douyin", "xhs", "wechat"}
	list := make([]Daemon, 0, len(order))

	for _, key := range order {
		spec, ok := specs[key]
		if !ok {
			continue
		}

		item := Daemon{
			Key:     spec.key,
			Name:    spec.name,
			Desc:    spec.desc,
			LogFile: filepath.Join(daemonLogDir, spec.logName+".out.log"),
		}

		if state, ok := states[key]; ok {
			item.Pid = state.Pid
			item.StartedAt = state.StartedAt
			item.Alive = processAlive(state.Pid)
		}

		if hb := heartbeatTime(spec.heartbeat); hb != "" {
			item.Heartbeat = hb
		}

		// 心跳在 2 分钟内更新过，就认为它在跑（脚本可能是电脑上手动起的，
		// 这时后台拿不到 PID，但心跳能证明存活）
		if !item.Alive && heartbeatFresh(spec.heartbeat) {
			item.Alive = true
		}

		// 状态文件里没有，但脚本可能是在电脑上手动起的（start-all.ps1），按命令行认领 PID
		if item.Pid == 0 && item.Alive {
			if pid := findPidByScript(spec.args[0]); pid > 0 {
				item.Pid = pid
				item.StartedAt = time.Now().Format("2006-01-02 15:04:05")
				states[key] = daemonState{Pid: pid, StartedAt: item.StartedAt}
				saveDaemonState(states)
			}
		}

		item.LastLog = lastLogLine(filepath.Join(daemonLogDir, spec.logName+".err.log"))

		list = append(list, item)
	}

	return list, nil
}

// StartDaemon 启动一个监听进程
func (leadService *LeadService) StartDaemon(key string) (string, error) {
	daemonMu.Lock()
	defer daemonMu.Unlock()

	specs := daemonSpecs()

	spec, ok := specs[key]
	if !ok {
		return "", fmt.Errorf("未知的进程：%s", key)
	}

	states := loadDaemonState()

	if state, ok := states[key]; ok && processAlive(state.Pid) {
		return "已经在运行了", nil
	}

	if err := os.MkdirAll(daemonLogDir, 0o755); err != nil {
		return "", err
	}

	outFile, err := os.OpenFile(filepath.Join(daemonLogDir, spec.logName+".out.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}

	errFile, err := os.OpenFile(filepath.Join(daemonLogDir, spec.logName+".err.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		outFile.Close()

		return "", err
	}

	cmd := exec.Command(spec.program, spec.args...)
	cmd.Dir = spec.workdir
	cmd.Stdout = outFile
	cmd.Stderr = errFile
	cmd.Env = append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"TEMP=E:\\ai\\data\\tmp",
		"TMP=E:\\ai\\data\\tmp",
	)

	if err := cmd.Start(); err != nil {
		outFile.Close()
		errFile.Close()

		return "", fmt.Errorf("启动失败：%w", err)
	}

	pid := cmd.Process.Pid

	states[key] = daemonState{Pid: pid, StartedAt: time.Now().Format("2006-01-02 15:04:05")}
	saveDaemonState(states)

	// 不 Wait，交给系统托管；进程退出后由状态查询判定
	go func() {
		_ = cmd.Wait()
		outFile.Close()
		errFile.Close()
	}()

	return fmt.Sprintf("已启动，PID %d", pid), nil
}

// StopDaemon 停止一个监听进程
func (leadService *LeadService) StopDaemon(key string) (string, error) {
	daemonMu.Lock()
	defer daemonMu.Unlock()

	specs := daemonSpecs()

	spec, ok := specs[key]
	if !ok {
		return "", fmt.Errorf("未知的进程：%s", key)
	}

	states := loadDaemonState()

	state, ok := states[key]

	if !ok || !processAlive(state.Pid) {
		// 没记录 PID，可能是电脑上手动起的，按命令行找一次
		pid := findPidByScript(spec.args[0])

		if pid <= 0 {
			delete(states, key)
			saveDaemonState(states)

			return "没找到进程，可能已经停了", nil
		}

		state = daemonState{Pid: pid}
	}

	kill := exec.Command(`C:\Windows\System32\taskkill.exe`, "/PID", strconv.Itoa(state.Pid), "/T", "/F")
	_ = kill.Run()

	delete(states, key)
	saveDaemonState(states)

	return "已停止", nil
}

func loadDaemonState() map[string]daemonState {
	result := map[string]daemonState{}

	raw, err := os.ReadFile(daemonStateFile)
	if err != nil {
		return result
	}

	_ = json.Unmarshal(raw, &result)

	return result
}

func saveDaemonState(states map[string]daemonState) {
	raw, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return
	}

	_ = os.MkdirAll(filepath.Dir(daemonStateFile), 0o755)
	_ = os.WriteFile(daemonStateFile, raw, 0o644)
}

// processAlive 用 tasklist 判断 PID 是否还活着（Windows 上 os.FindProcess 恒成功）
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	out, err := exec.Command(`C:\Windows\System32\tasklist.exe`, "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), strconv.Itoa(pid))
}

func heartbeatTime(path string) string {
	if path == "" {
		return ""
	}

	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	return info.ModTime().Format("2006-01-02 15:04:05")
}

// findPidByScript 按脚本名找进程，用于认领电脑上手动启动的监听进程
func findPidByScript(scriptPath string) int {
	name := filepath.Base(scriptPath)

	// 只匹配 node/python 进程，否则会匹配到正在执行这条查询的 powershell 自己
	ps := fmt.Sprintf(
		"Get-CimInstance Win32_Process | Where-Object { ($_.Name -eq 'node.exe' -or $_.Name -eq 'python.exe') -and $_.CommandLine -like '*%s*' } | Select-Object -First 1 -ExpandProperty ProcessId",
		name,
	)

	out, err := exec.Command(`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, "-NoProfile", "-NonInteractive", "-Command", ps).Output()

	debugLog(fmt.Sprintf("lookup %s -> out=%q err=%v", name, string(out), err))

	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || pid <= 0 {
		return 0
	}

	return pid
}
// lastLogLine 读日志最后一行非空内容，进程挂掉时用来看原因
func lastLogLine(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimRight(string(raw), "\r\n"), "\n")

	for i := len(lines) - 1; i >= 0 && i >= len(lines)-20; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		runes := []rune(line)
		if len(runes) > 120 {
			return string(runes[:120])
		}

		return line
	}

	return ""
}

func debugLog(msg string) {
	f, err := os.OpenFile(`E:\ai\data\logs\daemon-debug.log`, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}

	defer f.Close()

	_, _ = f.WriteString(time.Now().Format("15:04:05 ") + msg + "\n")
}

// heartbeatFresh 心跳文件是否在 2 分钟内更新过
func heartbeatFresh(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return time.Since(info.ModTime()) < 2*time.Minute
}
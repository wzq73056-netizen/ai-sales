package lead

import (
	"os"
	"strings"
	"time"
)

// WxLoginStatus 微信登录状态
//
// wx_watch.py 掉登录时会写 wx_key_failed.flag，之前没有任何地方读它，
// 结果就是「微信静默不回复，人以为在忙」。这里把它变成接口可见的状态。
type WxLoginStatus struct {
	Ok          bool   `json:"Ok"`
	FlagTime    string `json:"FlagTime"`
	LastSuccess string `json:"LastSuccess"`
	Message     string `json:"Message"`
}

const (
	wxKeyFailFlag    = `E:\ai\data\wx_key_failed.flag`
	wxHeartbeatFile  = `E:\ai\data\wx_heartbeat.txt`
	wxFlagWarnLimit  = 10 * time.Minute
	wxHeartbeatLimit = 3 * time.Minute
)

// GetWxLoginStatus 判断微信是否掉登录
func (leadService *LeadService) GetWxLoginStatus() WxLoginStatus {
	status := WxLoginStatus{Ok: true}

	if info, err := os.Stat(wxKeyFailFlag); err == nil {
		status.Ok = false
		status.FlagTime = info.ModTime().Format("2006-01-02 15:04:05")

		for {
			raw, readErr := os.ReadFile(wxKeyFailFlag)
			if readErr != nil {
				break
			}

			content := strings.TrimSpace(strings.TrimPrefix(string(raw), "\uFEFF"))
			if content != "" {
				status.FlagTime = content
			}

			break
		}

		status.Message = "微信掉登录了：请在工作机扫码登录，否则收不到消息、也不会自动回复"
	}

	if info, err := os.Stat(wxHeartbeatFile); err == nil {
		status.LastSuccess = info.ModTime().Format("2006-01-02 15:04:05")

		// 心跳断了也说明监听停了
		if time.Since(info.ModTime()) > wxHeartbeatLimit && status.Ok {
			status.Ok = false
			status.Message = "微信监听心跳超过 3 分钟没更新，检查监听进程是否在跑"
		}
	}

	return status
}

// GetSystemAlerts 系统级告警（首页红条用）
func (leadService *LeadService) GetSystemAlerts() []map[string]string {
	alerts := make([]map[string]string, 0, 3)

	login := leadService.GetWxLoginStatus()
	if !login.Ok {
		alerts = append(alerts, map[string]string{
			"Level":   "error",
			"Title":   "微信需要重新登录",
			"Detail":  login.Message,
			"Action":  "reLoginWx",
			"ActionName": "怎么处理",
		})
	}

	return alerts
}

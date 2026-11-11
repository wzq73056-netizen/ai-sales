package lead

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

func TestMergeRemark(t *testing.T) {
	cases := []struct {
		old  string
		mark string
		want string
	}{
		{"", "已转客户", "已转客户"},
		{"原备注", "已转客户", "原备注｜已转客户"},
		{"原备注｜已转客户", "已转客户", "原备注｜已转客户"},
		{"   ", "已转客户", "已转客户"},
	}

	for _, tc := range cases {
		if got := leadModel.MergeRemark(tc.old, tc.mark); got != tc.want {
			t.Fatalf("MergeRemark(%q, %q) = %q, want %q", tc.old, tc.mark, got, tc.want)
		}
	}
}

func TestNextRunTime(t *testing.T) {
	base := time.Date(2026, 11, 6, 10, 0, 0, 0, time.Local)

	task := &leadModel.DouyinTask{ScheduleType: "daily", Schedule: "03:30"}

	got := nextRunTime(task, base)
	if got != "2026-11-07 03:30:00" {
		t.Fatalf("daily 下次执行时间不对: %s", got)
	}

	// 同一天但时间还没到 → 当天
	early := time.Date(2026, 11, 6, 1, 0, 0, 0, time.Local)
	if got := nextRunTime(task, early); got != "2026-11-06 03:30:00" {
		t.Fatalf("当天未到点应排在当天: %s", got)
	}

	// 没有配置时不 panic
	empty := &leadModel.DouyinTask{}
	_ = nextRunTime(empty, base)
}

// 节流状态：暂停中应返回 true 和剩余分钟；已过期返回 false
func TestThrottlePaused(t *testing.T) {
	path := `E:\ai\data\scrape_throttle.json`
	backup, _ := os.ReadFile(path)
	defer func() {
		if backup != nil {
			os.WriteFile(path, backup, 0o644)
		} else {
			os.Remove(path)
		}
	}()

	// 未暂停
	writeThrottle(t, path, map[string]any{"level": 1, "pausedUntil": ""})
	if paused, _ := throttlePaused(); paused {
		t.Fatal("pausedUntil 为空时不该暂停")
	}

	// 暂停到 10 分钟后
	writeThrottle(t, path, map[string]any{
		"level":       2,
		"pausedUntil": time.Now().Add(10 * time.Minute).Format("2006-01-02 15:04:05"),
	})

	paused, minutes := throttlePaused()
	if !paused {
		t.Fatal("应处于暂停状态")
	}

	if minutes <= 0 || minutes > 11 {
		t.Fatalf("剩余分钟异常: %d", minutes)
	}

	// 已经过期
	writeThrottle(t, path, map[string]any{
		"level":       2,
		"pausedUntil": time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05"),
	})

	if paused, _ := throttlePaused(); paused {
		t.Fatal("暂停已过期不该再暂停")
	}
}

func writeThrottle(t *testing.T, path string, value map[string]any) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

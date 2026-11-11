package lead

import (
	"strings"
	"testing"

	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
)

func strp(v string) *string { return &v }
func intp(v int64) *int64 { return &v }

func TestGradeLead(t *testing.T) {
	cases := []struct {
		name      string
		lead      leadModel.Lead
		wantGrade string
		wantReady bool
	}{
		{
			name: "高分 + 电话 + 买方意图 + 城市 → S",
			lead: leadModel.Lead{
				Score:  intp(90),
				Phone:  strp("13800001111"),
				Remark: strp("求购鸡蛋 大量要货"),
				City:   strp("江苏"),
			},
			wantGrade: "S",
			wantReady: true,
		},
		{
			name: "中等分 + 微信 + 询价 → S（45+25+15=85）",
			lead: leadModel.Lead{
				Score:  intp(55),
				Wechat: strp("abc123"),
				Remark: strp("鸡蛋批发多少钱"),
			},
			wantGrade: "S",
			wantReady: true,
		},
		{
			name:      "低分 + 普通评论 → C（28 分）",
			lead:      leadModel.Lead{Score: intp(32), Remark: strp("这鸡蛋看着不错")},
			wantGrade: "C",
			wantReady: false,
		},
		{
			name:      "零分空内容 → D",
			lead:      leadModel.Lead{},
			wantGrade: "D",
			wantReady: false,
		},
		{
			name: "卖方词扣分：原本 A 掉到 B",
			lead: leadModel.Lead{
				Score:  intp(80),
				Remark: strp("厂家直销 鸡蛋批发供应"),
			},
			wantGrade: "B",
			wantReady: false,
		},
		{
			name: "有微信号即视为可联系",
			lead: leadModel.Lead{
				Score:  intp(50),
				Wechat: strp("wxid_abc"),
			},
			wantGrade: "A",
			wantReady: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			grade, reason, ready := GradeLead(&tc.lead)

			if grade != tc.wantGrade {
				t.Fatalf("等级不符: got %s want %s（理由 %s）", grade, tc.wantGrade, reason)
			}

			if ready != tc.wantReady {
				t.Fatalf("contact_ready 不符: got %v want %v", ready, tc.wantReady)
			}

			if reason == "" {
				t.Fatal("理由不该为空")
			}
		})
	}
}

// 同行词必须真的扣分：同样输入，去掉「供应」后等级更高
func TestGradeLeadPeerPenalty(t *testing.T) {
	withPeer := leadModel.Lead{Score: intp(80), Remark: strp("鸡蛋批发 供应")}
	withoutPeer := leadModel.Lead{Score: intp(80), Remark: strp("鸡蛋批发")}

	gradeWith, _, _ := GradeLead(&withPeer)
	gradeWithout, _, _ := GradeLead(&withoutPeer)

	order := map[string]int{"D": 0, "C": 1, "B": 2, "A": 3, "S": 4}

	if order[strings.TrimRight(gradeWith, "12")] >= order[strings.TrimRight(gradeWithout, "12")] {
		t.Fatalf("同行词应降低等级：同行=%s 非同行=%s", gradeWith, gradeWithout)
	}
}

// 评分越高等级不低于低评分
func TestGradeLeadMonotonic(t *testing.T) {
	order := map[string]int{"D": 0, "C": 1, "B": 2, "A": 3, "S": 4}
	prev := -1

	for _, score := range []int{0, 20, 35, 55, 80, 95} {
		lead := leadModel.Lead{Score: intp(int64(score))}
		grade, _, _ := GradeLead(&lead)

		level := order[strings.TrimRight(grade, "12")]

		if level < prev {
			t.Fatalf("评分 %d 的等级 %s 低于更低评分的等级", score, grade)
		}

		prev = level
	}
}

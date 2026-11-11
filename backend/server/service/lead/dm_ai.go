package lead

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	leadModel "github.com/flipped-aurora/gin-vue-admin/server/model/lead"
	kbfrag "github.com/flipped-aurora/gin-vue-admin/server/service/kbfrag"
)

// DmReplyRequest AI 生成私信回复的入参
type DmReplyRequest struct {
	Nick        string `json:"Nick"`
	LastMessage string `json:"LastMessage"`
	History     string `json:"History"`
	// Channel 对话发生在哪个平台：wechat / douyin / xhs
	Channel string `json:"Channel"`
	// Side 对方角色：buyer=买我们货的客户 / supplier=供货给我们的（空则按 buyer）
	Side string `json:"Side"`
}

// supplierRole 对方是供货商时用的人设：我们去采购
const supplierRole = `你是合肥一家餐饮企业的采购员「小李」，在联系鸡蛋批发商采购。
沟通目标：问清当日红蛋/粉蛋出场价、规格（中码/大码）、起订量、运费、结算方式（现结/月结），
条件合适时留下微信继续谈。
说话风格：口语、简短（30-60 字）、像真人打字，不用书面语，不列 1234，不自我介绍，不提自己是 AI。
如果对方已经报价或给了联系方式，就顺着确认或约下一步，不要重复问已经回答过的问题。`

// buyerRole 对方是客户时用的人设：我们卖货
const buyerRole = `你是合肥一家鸡蛋批发商的销售「小李」，在接待餐饮、超市、二批的采购。
沟通目标：先问清对方要什么（红蛋还是粉蛋、中码还是大码、每天要多少），
再报出规格、起订量、运费和结算方式（现结/月结），合适就约加微信或下单。
说话风格：口语、简短（30-60 字）、像真人打字，不用书面语，不列 1234，不自我介绍，不提自己是 AI。
如果对方已经说了需求，就直接给价和下一步，不要重复问已经回答过的问题。`

// defaultDmRole 默认角色设定（可通过 DM_ROLE_PROMPT 覆盖）
const defaultDmRole = `你是合肥一家餐饮企业的采购员「小李」，在抖音上联系鸡蛋批发商。
沟通目标：问清当日红蛋/粉蛋出场价、规格（中码/大码）、起订量、运费、结算方式（现结/月结），
条件合适时留下微信继续谈。
说话风格：口语、简短（30-60 字）、像真人打字，不用书面语，不列 1234，不自我介绍，不提自己是 AI。
如果对方已经报价或给了联系方式，就顺着确认或约下一步，不要重复问已经回答过的问题。
联系方式（重要）：线索最终要落到微信，所以在抖音/小红书里聊到有意向时，
主动要一次微信（例如"方便加个微信吗，我把规格和报价发表给你"），一次就够，对方不给就别追。
在微信里聊时不要再要微信，直接约下一步或要电话。
不要索要当前正在聊的这个平台（在抖音里不推抖音，在小红书里不推小红书）。`

// searchKnowledge 从知识库里检索与来信相关的片段（BM25 + LLM 精排）
func searchKnowledge(ctx context.Context, query string, topK int, side string) []string {
	svc := kbfrag.KnowledgeFragmentService{}

	// 默认用「鸡蛋批发采购知识库」(2002)，可用 DM_KB_ID 覆盖；0 表示检索全部
	// 按方向选知识库：客户问价用销售口径(2003)，我们采购用采购口径(2002)
	knowledgeId := int64(2003)
	envKey := "DM_KB_ID_SELLER"

	if strings.TrimSpace(side) == "supplier" {
		knowledgeId = 2002
		envKey = "DM_KB_ID_BUYER"
	}

	if v := os.Getenv(envKey); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			knowledgeId = n
		}
	}

	// 统一覆盖开关（调试用）
	if v := os.Getenv("DM_KB_ID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			knowledgeId = n
		}
	}

	hits, err := svc.SearchFragments(ctx, knowledgeId, query, topK, 0)
	if err != nil || len(hits) == 0 {
		return nil
	}

	out := make([]string, 0, len(hits))

	for _, hit := range hits {
		content := strings.TrimSpace(hit.Content)
		if content == "" {
			continue
		}
		if len([]rune(content)) > 400 {
			content = string([]rune(content)[:400])
		}

		label := hit.DocumentName
		if label == "" {
			label = "资料"
		}
		out = append(out, fmt.Sprintf("【%s】%s", label, content))
	}

	return out
}

// GenDmReply 用大模型根据对方最近消息生成一条回复
func (leadService *LeadService) GenDmReply(ctx context.Context, req DmReplyRequest) (string, error) {
	llm := loadLLMConfig()
	baseURL, apiKey := llm.BaseURL, llm.ApiKey
	if baseURL == "" || apiKey == "" {
		return "", fmt.Errorf("未配置大模型")
	}
	model := llm.Model
	if model == "" {
		model = "deepseek-chat"
	}

	// 调用方没指定方向时，按昵称查一下这条线索/客户属于哪一边
	if strings.TrimSpace(req.Side) == "" && strings.TrimSpace(req.Nick) != "" {
		req.Side = leadService.sideOf(req.Nick)
	}

	role := os.Getenv("DM_ROLE_PROMPT")
	if role == "" {
		// 两个方向的业务都要做：对方是客户就卖，对方是供货商就买
		if strings.TrimSpace(req.Side) == "supplier" {
			role = supplierRole
		} else {
			role = buyerRole
		}

		// 渠道不同，开头那句场景描述跟着变：在微信里还说"在抖音上联系"会很怪
		switch req.Channel {
		case "wechat":
			role = strings.Replace(role, "在抖音上联系鸡蛋批发商", "已经在微信上和对方聊着", 1)
			role = strings.Replace(role, "在接待餐饮、超市、二批的采购", "在微信上接待餐饮、超市、二批的采购", 1)
		case "xhs":
			role = strings.Replace(role, "在抖音上联系鸡蛋批发商", "在小红书上联系鸡蛋批发商", 1)
			role = strings.Replace(role, "在接待餐饮、超市、二批的采购", "在小红书上接待餐饮、超市、二批的采购", 1)
		}
	}

	role += "\n\n几条硬规则（必须遵守）：\n" +
		"1. 不要提自己是 AI、机器人、系统、助手，也不要说「我在学习」「我需要确认」这类机器口吻。\n" +
		"2. 对方发的语音已经转成文字放在下面，就当听清了直接回答内容，绝对不要说「听不清」「听不懂」「麻烦发文字」。\n" +
		"3. 不要重复上一句说过的话；同一个意思只说一遍。\n" +
		"4. 对方没给新信息时，别硬接一句，尽量推进一个具体问题（价格、规格、起订量、运费、结算）。"

	var sb strings.Builder
	sb.WriteString(role)
	sb.WriteString("\n\n")

	// 先检索知识库，把相关资料塞进提示词
	references := searchKnowledge(ctx, req.LastMessage, 3, req.Side)
	if len(references) > 0 {
		sb.WriteString("可参考资料（优先依据这些回答，资料没有的不要编造）：\n")
		for _, reference := range references {
			sb.WriteString("- ")
			sb.WriteString(reference)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if req.History != "" {
		sb.WriteString("最近对话：\n")
		sb.WriteString(req.History)
		sb.WriteString("\n\n")
	}

	fmt.Fprintf(&sb, "对方（%s）最新一条消息：%s\n\n", req.Nick, req.LastMessage)
	sb.WriteString("请只输出你要回复的那一句话，不要引号，不要任何解释。")

	payload, _ := json.Marshal(map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": sb.String()}},
		"temperature": 0.7,
	})

	client := &http.Client{Timeout: 60 * time.Second}
	httpReq, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("模型调用失败(%d)", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", fmt.Errorf("模型返回解析失败")
	}

	reply := strings.TrimSpace(parsed.Choices[0].Message.Content)
	reply = strings.Trim(reply, "\"'“”「」\n ")
	reply = strings.ReplaceAll(reply, "\n", " ")

	if len([]rune(reply)) > 200 {
		reply = string([]rune(reply)[:200])
	}

	if reply == "" {
		return "", fmt.Errorf("模型返回空内容")
	}

	return reply, nil
}

// sideOf 按昵称查对方角色：先看客户表，再看线索表
func (leadService *LeadService) sideOf(nick string) string {
	var customer leadModel.WxCustomer

	if err := global.GVA_DB.Where("nick_name = ?", nick).First(&customer).Error; err == nil {
		if customer.Side != "" {
			return customer.Side
		}
	}

	var leads []leadModel.Lead

	if err := global.GVA_DB.Where("company = ? OR contact_name = ?", nick, nick).Limit(1).Find(&leads).Error; err == nil && len(leads) > 0 {
		if leads[0].Side != nil {
			return strings.TrimSpace(*leads[0].Side)
		}
	}

	return ""
}
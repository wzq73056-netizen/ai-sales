package kbfrag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	kbfragModel "github.com/flipped-aurora/gin-vue-admin/server/model/kbfrag"
)

// SearchHit 一条检索结果
type SearchHit struct {
	Id           int64   `json:"Id"`
	KnowledgeId  int64   `json:"KnowledgeId"`
	DocumentId   int64   `json:"DocumentId"`
	DocumentName string  `json:"DocumentName"`
	Idx          int64   `json:"Idx"`
	DocId        string  `json:"DocId"`
	Content      string  `json:"Content"`
	Score        float64 `json:"Score"`
	Similarity   float64 `json:"Similarity"`
}

func derefI64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func derefStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// tokenize 中文 2-gram + 英文单词
func tokenize(text string) []string {
	var tokens []string
	var cnBuf []rune
	var enBuf []rune

	flushCn := func() {
		if len(cnBuf) == 0 {
			return
		}
		if len(cnBuf) == 1 {
			tokens = append(tokens, string(cnBuf))
		} else {
			for i := 0; i+1 < len(cnBuf); i++ {
				tokens = append(tokens, string(cnBuf[i:i+2]))
			}
		}
		cnBuf = cnBuf[:0]
	}
	flushEn := func() {
		if len(enBuf) == 0 {
			return
		}
		tokens = append(tokens, strings.ToLower(string(enBuf)))
		enBuf = enBuf[:0]
	}

	for _, r := range text {
		switch {
		case unicode.Is(unicode.Han, r):
			flushEn()
			cnBuf = append(cnBuf, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushCn()
			enBuf = append(enBuf, r)
		default:
			flushCn()
			flushEn()
		}
	}
	flushCn()
	flushEn()

	return tokens
}

// bm25Candidates BM25 粗排，返回候选（数量比最终 topK 大，供重排）
func bm25Candidates(fragments []kbfragModel.KnowledgeFragment, query string, limit int) []SearchHit {
	qTokens := tokenize(query)
	if len(qTokens) == 0 || len(fragments) == 0 {
		return nil
	}

	docTokens := make([][]string, len(fragments))
	df := map[string]int{}
	totalLen := 0

	for i, f := range fragments {
		ts := tokenize(derefStr(f.Content))
		docTokens[i] = ts
		totalLen += len(ts)

		seen := map[string]bool{}
		for _, t := range ts {
			if !seen[t] {
				seen[t] = true
				df[t]++
			}
		}
	}

	n := float64(len(fragments))
	avgLen := float64(totalLen) / n
	const k1, b = 1.5, 0.75

	hits := make([]SearchHit, 0, len(fragments))

	for i, f := range fragments {
		tf := map[string]int{}
		for _, t := range docTokens[i] {
			tf[t]++
		}

		score := 0.0
		for _, qt := range qTokens {
			freq := float64(tf[qt])
			if freq == 0 {
				continue
			}
			idf := math.Log(1 + (n-float64(df[qt])+0.5)/(float64(df[qt])+0.5))
			denom := freq + k1*(1-b+b*float64(len(docTokens[i]))/avgLen)
			score += idf * (freq * (k1 + 1)) / denom
		}

		if score > 0 {
			hits = append(hits, SearchHit{
				Id:          derefI64(f.Id),
				KnowledgeId: derefI64(f.KnowledgeId),
				DocumentId:  derefI64(f.DocumentId),
				Idx:         derefI64(f.Idx),
				DocId:       derefStr(f.DocId),
				Content:     derefStr(f.Content),
				Score:       score,
			})
		}
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })

	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}

	return hits
}

// rerankByLLM 用大模型给候选打相关性分（0-10），返回按分排序的结果
func rerankByLLM(ctx context.Context, query string, hits []SearchHit) ([]SearchHit, error) {
	baseURL := os.Getenv("RAG_CHAT_BASE_URL")
	apiKey := os.Getenv("RAG_CHAT_API_KEY")
	if baseURL == "" || apiKey == "" || len(hits) == 0 {
		return hits, nil
	}

	model := os.Getenv("RAG_CHAT_MODEL")
	if model == "" {
		model = "deepseek-chat"
	}

	var sb strings.Builder
	sb.WriteString("你是检索结果相关性评审。给定用户问题，为每个候选片段打 0-10 分：\n")
	sb.WriteString("10=直接回答问题，7-9=高度相关，4-6=部分相关，1-3=勉强相关，0=无关。\n")
	sb.WriteString("只输出 JSON 数组，形如 [{\"i\":1,\"s\":9}]，不要解释。\n\n")
	fmt.Fprintf(&sb, "问题：%s\n\n", query)

	for i, h := range hits {
		text := h.Content
		if len([]rune(text)) > 300 {
			text = string([]rune(text)[:300])
		}
		fmt.Fprintf(&sb, "[%d] %s\n\n", i+1, text)
	}

	payload, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": sb.String()},
		},
		"temperature": 0,
	})

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return hits, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return hits, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return hits, fmt.Errorf("rerank 调用失败(%d)", resp.StatusCode)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return hits, fmt.Errorf("rerank 解析失败")
	}

	raw := parsed.Choices[0].Message.Content
	raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "```json"))
	raw = strings.TrimSuffix(strings.TrimSpace(raw), "```")
	raw = strings.TrimSpace(raw)

	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start < 0 || end <= start {
		return hits, fmt.Errorf("rerank 返回格式异常")
	}

	var scores []struct {
		I int     `json:"i"`
		S float64 `json:"s"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &scores); err != nil {
		return hits, err
	}

	scoreMap := map[int]float64{}
	for _, item := range scores {
		scoreMap[item.I] = item.S
	}

	reranked := make([]SearchHit, 0, len(hits))
	for i, h := range hits {
		s, ok := scoreMap[i+1]
		if !ok {
			s = 0
		}
		h.Similarity = s * 10 // 0-10 → 0-100
		if h.Similarity > 0 {
			reranked = append(reranked, h)
		}
	}

	sort.Slice(reranked, func(i, j int) bool { return reranked[i].Similarity > reranked[j].Similarity })

	return reranked, nil
}

// SearchFragments 检索：BM25 粗排 → LLM 精排 → 阈值过滤
func (kbfragService *KnowledgeFragmentService) SearchFragments(ctx context.Context, knowledgeId int64, query string, topK int, minScore float64) ([]SearchHit, error) {
	var fragments []kbfragModel.KnowledgeFragment

	db := global.GVA_DB.WithContext(ctx).Model(&kbfragModel.KnowledgeFragment{})
	if knowledgeId != 0 {
		db = db.Where("knowledge_id = ?", knowledgeId)
	}
	if err := db.Find(&fragments).Error; err != nil {
		return nil, err
	}

	if topK <= 0 {
		topK = 5
	}

	// 粗排取 30 条候选
	candidates := bm25Candidates(fragments, query, 30)
	if len(candidates) == 0 {
		return []SearchHit{}, nil
	}

	// 精排
	ranked, err := rerankByLLM(ctx, query, candidates)
	if err != nil || len(ranked) == 0 {
		// 大模型不可用/全部打 0 分时，退回 BM25 归一化分数，避免整页空白
		maxScore := ranked0(candidates)
		for i := range candidates {
			if maxScore > 0 {
				candidates[i].Similarity = math.Round(candidates[i].Score/maxScore*10000) / 100
			}
		}
		ranked = candidates
	}

	// 补文档名
	docNames := map[int64]string{}
	var docs []kbfragModel.KnowledgeDocument
	if err := global.GVA_DB.WithContext(ctx).Find(&docs).Error; err == nil {
		for _, d := range docs {
			docNames[d.Id] = d.Name
		}
	}

	out := make([]SearchHit, 0, topK)
	for _, h := range ranked {
		if h.DocumentId != 0 {
			h.DocumentName = docNames[h.DocumentId]
		}
		if h.DocumentName == "" {
			// 直接导入的片段没有文档记录，退回用卡片/文档编号当来源标识
			if h.DocId != "" {
				h.DocumentName = h.DocId
			} else {
				h.DocumentName = "手工录入"
			}
		}
		h.Score = math.Round(h.Score*10000) / 10000

		if h.Similarity < minScore {
			continue
		}
		out = append(out, h)
		if len(out) >= topK {
			break
		}
	}

	return out, nil
}

func ranked0(hits []SearchHit) float64 {
	max := 0.0
	for _, h := range hits {
		if h.Score > max {
			max = h.Score
		}
	}
	return max
}

package kbfrag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ChatResult 一次 RAG 问答的结果
type ChatResult struct {
	// retrieval-only：只检索不生成（没配置大模型）
	// llm：调用大模型生成
	Mode    string      `json:"Mode"`
	Answer  string      `json:"Answer"`
	Prompt  string      `json:"Prompt"`
	Sources []SearchHit `json:"Sources"`
}

// buildPrompt 把检索到的分片拼成 RAG 提示词
func buildPrompt(question string, hits []SearchHit) string {
	var sb strings.Builder

	sb.WriteString("你是知识库助手。请只依据下面的资料回答，资料里没有的内容要明确说不知道。\n\n")

	if len(hits) == 0 {
		sb.WriteString("（没有检索到相关资料）\n")
	} else {
		sb.WriteString("资料：\n")
		for i, h := range hits {
			fmt.Fprintf(&sb, "[%d] %s\n", i+1, h.Content)
		}
	}

	fmt.Fprintf(&sb, "\n问题：%s\n", question)

	return sb.String()
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Chat 检索 + 生成（RAG）。没有配置大模型时只返回检索结果和将要发送的提示词。
func (kbfragService *KnowledgeFragmentService) Chat(ctx context.Context, knowledgeId int64, question string, topK int) (ChatResult, error) {
	hits, err := kbfragService.SearchFragments(ctx, knowledgeId, question, topK, 0)
	if err != nil {
		return ChatResult{}, err
	}

	prompt := buildPrompt(question, hits)
	result := ChatResult{Mode: "retrieval-only", Prompt: prompt, Sources: hits}

	baseURL := os.Getenv("RAG_CHAT_BASE_URL")
	apiKey := os.Getenv("RAG_CHAT_API_KEY")

	if baseURL == "" || apiKey == "" {
		return result, nil
	}

	model := os.Getenv("RAG_CHAT_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	payload, _ := json.Marshal(chatRequest{
		Model:    model,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
	})

	client := &http.Client{Timeout: 60 * time.Second}

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("模型调用失败(%d): %s", resp.StatusCode, string(body[:min(len(body), 300)]))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return result, err
	}

	if len(parsed.Choices) > 0 {
		result.Answer = parsed.Choices[0].Message.Content
		result.Mode = "llm"
	}

	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

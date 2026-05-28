package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIProvider 调 OpenAI 兼容的 /v1/chat/completions
type OpenAIProvider struct {
	BaseURL string
	APIKey  string
	Model   string

	HTTPClient *http.Client // 默认 http.DefaultClient
}

// NewOpenAIProvider 构造器
func NewOpenAIProvider(baseURL, apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{BaseURL: baseURL, APIKey: apiKey, Model: model}
}

// Stream 以 stream=true 发起 chat completion，按 SSE 推送 delta。
func (p *OpenAIProvider) Stream(ctx context.Context, req Request) (<-chan Chunk, error) {
	body, err := buildRequestBody(req, p.Model)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("post chat completions: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chat completions: status %d: %s", resp.StatusCode, string(b))
	}

	ch := make(chan Chunk, 16)
	go streamSSE(ctx, resp.Body, ch)
	return ch, nil
}

// streamSSE 按行扫描 SSE body，解析 `data: {...}` 推到 ch。
// 在 ctx 取消 / `[DONE]` / EOF 时退出并 close(ch)。
func streamSSE(ctx context.Context, body io.ReadCloser, ch chan<- Chunk) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 4096), 1<<20)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			select {
			case <-ctx.Done():
			case ch <- Chunk{Done: true}:
			}
			return
		}

		var delta openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &delta); err != nil {
			continue // 跳过非法 JSON
		}
		for _, choice := range delta.Choices {
			text := choice.Delta.Content
			if text == "" {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ch <- Chunk{Text: text}:
			}
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case <-ctx.Done():
		case ch <- Chunk{Err: err}:
		}
	}
}

// buildRequestBody 拼接 chat completions 请求体 JSON。
func buildRequestBody(req Request, defaultModel string) ([]byte, error) {
	model := req.Model
	if model == "" {
		model = defaultModel
	}
	msgs := make([]openAIMessage, 0, 2)
	if req.System != "" {
		msgs = append(msgs, openAIMessage{Role: "system", Content: req.System})
	}
	if req.User != "" {
		msgs = append(msgs, openAIMessage{Role: "user", Content: req.User})
	}
	body := openAIChatRequest{
		Model:       model,
		Messages:    msgs,
		Temperature: req.Temperature,
		Stream:      true,
	}
	// 有 JSONSchema 时启用 JSON 输出模式
	if req.JSONSchema != nil {
		body.ResponseFormat = &openAIResponseFormat{Type: "json_object"}
	}
	return json.Marshal(body)
}

type openAIChatRequest struct {
	Model          string                `json:"model"`
	Messages       []openAIMessage       `json:"messages"`
	Temperature    float32               `json:"temperature,omitempty"`
	Stream         bool                  `json:"stream"`
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFormat struct {
	Type string `json:"type"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

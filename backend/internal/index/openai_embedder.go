package index

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIEmbedder 调 OpenAI 兼容的 /v1/embeddings。
// 推荐用 text-embedding-3-small（1536 维，$0.02/M tokens，中文质量够 demo）。
// DeepSeek 没 embedding API；要么用 OpenAI 真账号，要么换豆包 doubao-embedding / Voyage。
type OpenAIEmbedder struct {
	BaseURL    string
	APIKey     string
	Model      string // 空时默认 text-embedding-3-small
	HTTPClient *http.Client
}

// NewOpenAIEmbedder 构造器
func NewOpenAIEmbedder(baseURL, apiKey, model string) *OpenAIEmbedder {
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIEmbedder{BaseURL: baseURL, APIKey: apiKey, Model: model}
}

// Embed 批量请求；OpenAI 一次最多 2048 条；超过分批。
// 失败时整批返 err，不部分返回 —— 调用方应重试或降级 Noop。
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	const batchLimit = 2048
	out := make([][]float32, 0, len(texts))
	for i := 0; i < len(texts); i += batchLimit {
		end := min(i+batchLimit, len(texts))
		batch, err := e.embedBatch(ctx, texts[i:end])
		if err != nil {
			return nil, fmt.Errorf("embed batch [%d,%d): %w", i, end, err)
		}
		out = append(out, batch...)
	}
	return out, nil
}

func (e *OpenAIEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	body, err := json.Marshal(openAIEmbeddingsRequest{Model: e.Model, Input: texts})
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(e.BaseURL, "/") + "/v1/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.APIKey)

	client := e.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post embeddings: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embeddings: status %d: %s", resp.StatusCode, string(b))
	}

	var parsed openAIEmbeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode embeddings: %w", err)
	}
	// OpenAI 返回顺序与 input 一一对应（按 index 排）
	out := make([][]float32, len(texts))
	for _, d := range parsed.Data {
		if d.Index < 0 || d.Index >= len(out) {
			continue
		}
		out[d.Index] = d.Embedding
	}
	return out, nil
}

type openAIEmbeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingsResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

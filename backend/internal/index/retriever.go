// Package index 是 v2 RAG / 代码图谱的入口。
// v1 用 NoopRetriever 占位；v2 换实现，调用方签名不动。
package index

import "context"

// Reference 一条检索到的代码片段，作为 L4 上下文注入 prompt。
type Reference struct {
	File    string // 源文件路径
	Snippet string // 检索命中的代码 / 文档片段
	Reason  string // 为什么命中（"defines X" / "calls Y"）
}

// Retriever 按 query 召回最多 k 条 Reference。
type Retriever interface {
	Retrieve(ctx context.Context, query string, k int) ([]Reference, error)
}

// Embedder 批量产出文本向量。
// 与 chat Provider 分开，方便挑专门的 embedding 模型。
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// NoopRetriever 永远返回空；v1 默认注入。
type NoopRetriever struct{}

// Retrieve 实现 Retriever。
func (NoopRetriever) Retrieve(_ context.Context, _ string, _ int) ([]Reference, error) {
	return nil, nil
}

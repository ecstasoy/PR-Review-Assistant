// Package index 是 v2 RAG / 代码图谱的入口。
// v1 用 NoopRetriever 占位；v2 换实现，调用方签名不动。
package index

import "context"

// Reference 一条检索到的代码片段，作为 L4 上下文注入 prompt。
type Reference struct {
	File    string // 源文件路径
	Snippet string // 检索命中的代码 / 文档片段
	Reason  string // 为什么命中（"defines X" / "calls Y" / "cosine=0.83"）
}

// Retriever 按 scope + query 召回最多 k 条 Reference。
// scope 是命名空间（如 "owner/repo"），避免跨仓库串扰；空 scope 不限。
type Retriever interface {
	Retrieve(ctx context.Context, scope, query string, k int) ([]Reference, error)
}

// Embedder 批量产出文本向量。
// 与 chat Provider 分开，方便挑专门的 embedding 模型。
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// IndexerChunk 单个待索引文本片段；与 SQLiteRetriever.Chunk 同形状。
// 抽到 interface 包让 Indexer 接口不依赖具体实现。
type IndexerChunk struct {
	Path    string
	Idx     int    // 同 path 下的序号；切大文件用
	Content string // 实际文本内容
}

// Indexer 把 chunks 写入索引；与 Retriever 解耦读写。
// SQLiteRetriever 同时实现两个接口；生产可拆给独立 worker（v3 异步索引）。
type Indexer interface {
	UpsertMany(ctx context.Context, scope string, chunks []IndexerChunk) error
}

// NoopIndexer Indexer 的空实现；RAG 关闭时注入避免 nil 检查。
type NoopIndexer struct{}

// UpsertMany 不做任何事。
func (NoopIndexer) UpsertMany(_ context.Context, _ string, _ []IndexerChunk) error {
	return nil
}

// NoopRetriever 永远返回空；v1 默认注入。
type NoopRetriever struct{}

// Retrieve 实现 Retriever。
func (NoopRetriever) Retrieve(_ context.Context, _, _ string, _ int) ([]Reference, error) {
	return nil, nil
}

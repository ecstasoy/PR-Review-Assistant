package index

import (
	"context"
	"hash/fnv"
	"math"
)

// MockEmbedder 确定性 hash → 向量；同文本同输出，方便 CI + 演示无 key 跑通。
// 不真实反映语义相似度（hash 衍生），仅用于「pipeline 跑通」级测试。
type MockEmbedder struct {
	Dim int // 维度；默认 1536 与 text-embedding-3-small 对齐
}

// NewMockEmbedder 默认 1536 维
func NewMockEmbedder() *MockEmbedder { return &MockEmbedder{Dim: 1536} }

// Embed 单线程 / 同步；批量大小不限
func (m *MockEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	dim := m.Dim
	if dim <= 0 {
		dim = 1536
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = hashToVec(t, dim)
	}
	return out, nil
}

// hashToVec 把字符串 FNV-hash 成 [dim] float32；用余数 → 三角函数生成稳定模式。
// 不追求统计性质，仅保证「不同文本 → 不同向量」+「相同文本 → 同向量」+ L2 归一化（与 OpenAI 输出对齐）。
func hashToVec(s string, dim int) []float32 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	seed := h.Sum64()
	out := make([]float32, dim)
	var sumSq float64
	for i := range dim {
		v := math.Sin(float64(seed+uint64(i)*1009)) // 1009 质数让相邻 dim 解耦
		out[i] = float32(v)
		sumSq += v * v
	}
	norm := math.Sqrt(sumSq)
	if norm > 0 {
		for i := range out {
			out[i] = float32(float64(out[i]) / norm)
		}
	}
	return out
}

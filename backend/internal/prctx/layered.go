package prctx

import (
	"fmt"
	"strings"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/index"
)

const (
	// DefaultTokenLimit 默认 token 上限（DeepSeek 64K 输入 - 16K 输出预留）
	DefaultTokenLimit = 48000

	// charsPerToken 粗略估算：中文 ~2 char/token，英文 ~4 char/token，取保守均值 3
	charsPerToken = 3

	// floorL2Tokens L2 至少保留的预算，避免大 L3 / L1 把 L2 挤光
	floorL2Tokens = 1000
)

// LayeredBuilder 实现 Builder，按 L1:L2:L3 = 4:5:1 分配 token 预算，
// 超限时按 L3 → L2 → L1 顺序压缩。
type LayeredBuilder struct {
	TokenLimit int
	Retriever  index.Retriever // v1 注入 NoopRetriever；v2 RAG 填 L4
}

// Option 构造 LayeredBuilder 的可选参数。
type Option func(*LayeredBuilder)

// WithTokenLimit 覆盖默认 token 上限。
func WithTokenLimit(n int) Option {
	return func(b *LayeredBuilder) { b.TokenLimit = n }
}

// WithRetriever 注入 RAG 检索器（v1 默认 NoopRetriever）。
func WithRetriever(r index.Retriever) Option {
	return func(b *LayeredBuilder) { b.Retriever = r }
}

// NewLayeredBuilder 构造一个带默认值的 LayeredBuilder。
func NewLayeredBuilder(opts ...Option) *LayeredBuilder {
	b := &LayeredBuilder{
		TokenLimit: DefaultTokenLimit,
		Retriever:  index.NoopRetriever{},
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Build 按预算分配 + 压缩生成 Context。
func (b *LayeredBuilder) Build(pr github.PullRequest) (Context, error) {
	if b.TokenLimit <= 0 {
		return Context{}, fmt.Errorf("token limit must be positive: %d", b.TokenLimit)
	}

	// L1：PR meta + per-file 加减行统计，体量小，永远全留
	l1Str := buildL1Meta(pr)
	l1Tokens := estimateTokens(l1Str)
	if l1Tokens > b.TokenLimit {
		return Context{}, fmt.Errorf("L1 meta exceeds token limit: used=%d limit=%d", l1Tokens, b.TokenLimit)
	}

	// L3：约定文件（v1 Fetcher 暂不填 Conventions，这里安全处理空值）
	l3StrFull := buildL3Conventions(pr.Conventions)
	l3Budget := b.TokenLimit / 10 // 10% 预算
	maxL3Budget := b.TokenLimit - l1Tokens - floorL2Tokens
	if maxL3Budget < 0 {
		maxL3Budget = 0
	}
	if l3Budget > maxL3Budget {
		l3Budget = maxL3Budget
	}

	l3Str := l3StrFull
	if l3Budget == 0 {
		l3Str = ""
	} else if estimateTokens(l3Str) > l3Budget {
		l3Str = truncate(l3Str, l3Budget*charsPerToken)
	}
	l3Tokens := estimateTokens(l3Str)

	// L2 可用预算 = 总 - L1 - L3（严格不超出 TokenLimit）
	l2Avail := b.TokenLimit - l1Tokens - l3Tokens
	if l2Avail < 0 {
		l2Avail = 0
	}

	// L2 按 patch 大小逐个塞入；超出预算的文件入 Dropped 列表
	l2Files, l2Used, dropped := allocateL2(pr.Files, l2Avail)
	return Context{
		L1Meta:        l1Str,
		L2Files:       l2Files,
		L3Conventions: l3Str,
		L4References:  nil, // v2 RAG 接入位
		BudgetReport: BudgetReport{
			TokenLimit: b.TokenLimit,
			UsedL1:     l1Tokens,
			UsedL2:     l2Used,
			UsedL3:     l3Tokens,
			Dropped:    dropped,
		},
	}, nil
}

func buildL1Meta(pr github.PullRequest) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "仓库: %s/%s#%d\n", pr.Owner, pr.Repo, pr.Number)
	fmt.Fprintf(&sb, "标题: %s\n", pr.Title)
	if pr.Body != "" {
		fmt.Fprintf(&sb, "正文:\n%s\n", pr.Body)
	}
	fmt.Fprintf(&sb, "改动 %d 个文件：\n", len(pr.Files))
	for _, f := range pr.Files {
		fmt.Fprintf(&sb, "- %s (%s) +%d -%d\n", f.Path, f.Status, f.Additions, f.Deletions)
	}
	return sb.String()
}

func buildL3Conventions(c github.Conventions) string {
	var parts []string
	if c.Readme != "" {
		parts = append(parts, "## README\n"+c.Readme)
	}
	if c.Contributing != "" {
		parts = append(parts, "## CONTRIBUTING\n"+c.Contributing)
	}
	if c.AgentDocs != "" {
		parts = append(parts, "## AGENT DOCS\n"+c.AgentDocs)
	}
	return strings.Join(parts, "\n\n")
}

func allocateL2(files []github.File, budget int) ([]FileContext, int, []string) {
	out := make([]FileContext, 0, len(files))
	dropped := []string{}
	used := 0
	for _, f := range files {
		patchTokens := estimateTokens(f.Patch)
		if used+patchTokens > budget {
			dropped = append(dropped, f.Path)
			continue
		}
		out = append(out, FileContext{
			Path:  f.Path,
			Patch: f.Patch,
		})
		used += patchTokens
	}
	return out, used, dropped
}

// estimateTokens 粗略估 token 数（不引 tiktoken 等真实分词器，按字符数 / 3 近似）。
func estimateTokens(s string) int {
	return (len(s) + charsPerToken - 1) / charsPerToken
}

// truncate 截断字符串到指定字符数，末尾加省略标记。
func truncate(s string, maxChars int) string {
	if len(s) <= maxChars {
		return s
	}
	return s[:maxChars] + "\n...[truncated]"
}

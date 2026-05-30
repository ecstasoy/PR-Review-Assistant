package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
)

// 三个 builtin tool：read_file / list_dir / grep_patches
//
// 安全设计（沙盒）：所有 tool 的 file 参数必须落在 cachedFiles 内，
// 不能逃出 PR 改动集 —— 防止 LLM 通过引导任意读取外部 / 系统文件。
// 文件集对应 review 那一刻的 cached payload，跟 prctx.Context.L2Files 同源。

// NewReadFileTool 读单个改动文件的 patch hunk + 受限元数据。
// args: { "file": "path/to/file" }
// 返回：patch + status + +/- 行数；超出沙盒返 error。
func NewReadFileTool(files []gh.File) Tool {
	byPath := indexByPath(files)
	return &simpleTool{
		spec: ToolSpec{
			Name:        "read_file",
			Description: "读取 PR 内某个改动文件的 diff hunk 和 add/delete 行数。仅限本次 PR 改动集",
			Parameters:  json.RawMessage(`{"type":"object","required":["file"],"properties":{"file":{"type":"string","description":"PR 内某改动文件的相对路径"}}}`),
		},
		run: func(_ context.Context, raw json.RawMessage) (string, error) {
			var a struct {
				File string `json:"file"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", fmt.Errorf("read_file: 参数解析失败: %w", err)
			}
			if a.File == "" {
				return "", errors.New("read_file: file 不能为空")
			}
			f, ok := byPath[a.File]
			if !ok {
				return "", fmt.Errorf("read_file: %q 不在本 PR 改动集（沙盒拒绝）", a.File)
			}
			return fmt.Sprintf("file: %s\nstatus: %s\n+%d -%d\n\npatch:\n%s",
				f.Path, f.Status, f.Additions, f.Deletions, f.Patch), nil
		},
	}
}

// NewListDirTool 列出某目录下的改动文件；prefix 空 = 仓库根
// args: { "prefix": "src/" }
// 返回：每行一个 file path + status；同样限沙盒内
func NewListDirTool(files []gh.File) Tool {
	return &simpleTool{
		spec: ToolSpec{
			Name:        "list_dir",
			Description: "列出 PR 中某目录下的改动文件。prefix 为空时列所有",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"prefix":{"type":"string","description":"目录前缀，如 \"src/\"；为空时列所有改动"}}}`),
		},
		run: func(_ context.Context, raw json.RawMessage) (string, error) {
			var a struct {
				Prefix string `json:"prefix"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", fmt.Errorf("list_dir: 参数解析失败: %w", err)
			}
			prefix := strings.TrimSpace(a.Prefix)
			matched := make([]gh.File, 0, len(files))
			for _, f := range files {
				if prefix == "" || strings.HasPrefix(path.Clean(f.Path), prefix) {
					matched = append(matched, f)
				}
			}
			sort.Slice(matched, func(i, j int) bool { return matched[i].Path < matched[j].Path })
			if len(matched) == 0 {
				return fmt.Sprintf("（沙盒内 prefix=%q 无改动文件）", prefix), nil
			}
			var sb strings.Builder
			for _, f := range matched {
				fmt.Fprintf(&sb, "%-9s +%-4d -%-4d  %s\n", f.Status, f.Additions, f.Deletions, f.Path)
			}
			return sb.String(), nil
		},
	}
}

// NewGrepTool 在所有 cached patches 上 grep 字符串或正则。
// args: { "pattern": "TODO", "regex": false (默认), "max_matches": 20 (默认) }
// 返回：每条命中显示 file:line: 上下文行
func NewGrepTool(files []gh.File) Tool {
	return &simpleTool{
		spec: ToolSpec{
			Name:        "grep_patches",
			Description: "在 PR diff 内 grep 字符串或正则；返回 file:patch行号: 命中行（行号为 patch 内序号，非原文件行号）",
			Parameters:  json.RawMessage(`{"type":"object","required":["pattern"],"properties":{"pattern":{"type":"string"},"regex":{"type":"boolean","description":"true 时按正则编译，否则字面匹配"},"max_matches":{"type":"integer","description":"最多返回多少条命中，默认 20"}}}`),
		},
		run: func(_ context.Context, raw json.RawMessage) (string, error) {
			var a struct {
				Pattern    string `json:"pattern"`
				Regex      bool   `json:"regex"`
				MaxMatches int    `json:"max_matches"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", fmt.Errorf("grep_patches: 参数解析失败: %w", err)
			}
			if a.Pattern == "" {
				return "", errors.New("grep_patches: pattern 不能为空")
			}
			if a.MaxMatches <= 0 {
				a.MaxMatches = 20
			}
			var matcher func(string) bool
			if a.Regex {
				re, err := regexp.Compile(a.Pattern)
				if err != nil {
					return "", fmt.Errorf("grep_patches: 非法正则: %w", err)
				}
				matcher = re.MatchString
			} else {
				p := a.Pattern
				matcher = func(s string) bool { return strings.Contains(s, p) }
			}
			var hits []string
			truncated := false
			for _, f := range files {
				if f.Patch == "" {
					continue
				}
				for i, line := range strings.Split(f.Patch, "\n") {
					if matcher(line) {
						hits = append(hits, fmt.Sprintf("%s:%d: %s", f.Path, i+1, line))
						if len(hits) >= a.MaxMatches {
							truncated = true
							goto done
						}
					}
				}
			}
		done:
			if len(hits) == 0 {
				return fmt.Sprintf("（pattern=%q 在 %d 个 patch 中无命中）", a.Pattern, len(files)), nil
			}
			out := strings.Join(hits, "\n")
			if truncated {
				out += fmt.Sprintf("\n（已截断到 max_matches=%d）", a.MaxMatches)
			}
			return out, nil
		},
	}
}

// RegisterDefaults 把三个 builtin tool 都注册到 r。
// 调用方在每次 review steer / agent 循环开始时构造 Registry，
// 用本 PR 的 files 绑定沙盒边界。
func RegisterDefaults(r *Registry, files []gh.File) {
	r.Register(NewReadFileTool(files))
	r.Register(NewListDirTool(files))
	r.Register(NewGrepTool(files))
}

// simpleTool ToolSpec + run 函数的轻量包装；
// 避免每个 builtin tool 都要写一个完整 struct + 方法集
type simpleTool struct {
	spec ToolSpec
	run  func(ctx context.Context, args json.RawMessage) (string, error)
}

func (s *simpleTool) Spec() ToolSpec { return s.spec }
func (s *simpleTool) Run(ctx context.Context, args json.RawMessage) (string, error) {
	return s.run(ctx, args)
}

func indexByPath(files []gh.File) map[string]gh.File {
	m := make(map[string]gh.File, len(files))
	for _, f := range files {
		m[f.Path] = f
	}
	return m
}

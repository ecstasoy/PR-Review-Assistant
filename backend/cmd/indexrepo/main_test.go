package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitToWindows_ShortFile(t *testing.T) {
	got := splitToWindows("hello world")
	if len(got) != 1 || got[0] != "hello world" {
		t.Fatalf("short file 应单 chunk 原样返回，got %+v", got)
	}
}

func TestSplitToWindows_LongFile(t *testing.T) {
	// 10K 字符 → 至少 4 个 chunk (3K-overlap=2.7K step)
	src := strings.Repeat("a\n", 5000) // 10000 字符
	got := splitToWindows(src)
	if len(got) < 3 {
		t.Fatalf("expected ≥3 chunks for 10K, got %d", len(got))
	}
	// 每个 chunk 不超过 chunkChars + 行尾回溯余量
	for i, c := range got {
		if len(c) > chunkChars+100 {
			t.Errorf("chunk[%d] too big: %d > %d", i, len(c), chunkChars)
		}
	}
	// 拼回去应该 ≥ 原长（含重叠）；不应短于
	totalLen := 0
	for _, c := range got {
		totalLen += len(c)
	}
	if totalLen < len(src) {
		t.Errorf("total chunks chars %d < src %d，应至少持平", totalLen, len(src))
	}
}

func TestSplitToWindows_BreaksAtNewline(t *testing.T) {
	// 用 typical 80 字符代码行；回溯窗 100 字符总能找到 \n
	line := strings.Repeat("x", 79) + "\n"
	src := strings.Repeat(line, 200) // 16K 字符
	got := splitToWindows(src)
	if len(got) < 3 {
		t.Fatalf("expected ≥3 chunks, got %d", len(got))
	}
	for i, c := range got[:len(got)-1] { // 最后一个 chunk 可能到 EOF 不一定换行
		if !strings.HasSuffix(c, "\n") {
			t.Errorf("chunk[%d] 末尾应在行边界（含\\n），got 末 30 字符 = %q", i, c[max(0, len(c)-30):])
		}
	}
}

func TestIsLikelyText(t *testing.T) {
	cases := []struct {
		name string
		b    []byte
		want bool
	}{
		{"plain ascii", []byte("hello world"), true},
		{"utf8 chinese", []byte("你好世界"), true},
		{"empty", []byte{}, true},
		{"binary with null", []byte{0x00, 0x01, 0x02}, false},
		{"null in middle", append([]byte("abc"), 0x00), false},
	}
	for _, tc := range cases {
		if got := isLikelyText(tc.b); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestScanRepo_SkipsBinaryAndUnwanted(t *testing.T) {
	dir := t.TempDir()

	// 应索引
	mustWrite(t, filepath.Join(dir, "a.go"), "package main\nfunc main() {}\n")
	mustWrite(t, filepath.Join(dir, "README.md"), "# hi")
	mustWrite(t, filepath.Join(dir, "sub/util.ts"), "export const x = 1")

	// 应跳过：扩展名不在白名单
	mustWrite(t, filepath.Join(dir, "image.png"), "binary-ish but ext skipped")
	// 应跳过：node_modules 整个目录
	mustWrite(t, filepath.Join(dir, "node_modules/pkg/index.js"), "module.exports = {}")
	// 应跳过：lock 文件
	mustWrite(t, filepath.Join(dir, "package-lock.json"), "{}")
	// 应跳过：含 null byte 的"伪文本"
	mustWrite(t, filepath.Join(dir, "weird.go"), "package main\x00\nfunc x(){}")

	chunks, files, skipped, err := scanRepo(dir)
	if err != nil {
		t.Fatalf("scanRepo: %v", err)
	}
	if files != 3 {
		t.Errorf("应索引 3 个文件（a.go README.md util.ts），got %d", files)
	}
	if skipped < 3 {
		t.Errorf("应跳过至少 3 个，got %d", skipped)
	}
	// 验证 path 归一化为 /
	for _, c := range chunks {
		if strings.Contains(c.Path, `\`) {
			t.Errorf("path 含反斜杠未归一化: %s", c.Path)
		}
		if c.PRNumber != 0 {
			t.Errorf("离线索引 PRNumber 应为 0，got %d", c.PRNumber)
		}
	}
	// 应含 sub/util.ts 验证目录递归
	var seen []string
	for _, c := range chunks {
		seen = append(seen, c.Path)
	}
	if !containsString(seen, "sub/util.ts") {
		t.Errorf("未递归到 sub/util.ts: %v", seen)
	}
	if containsString(seen, "weird.go") {
		t.Errorf("含 null byte 的伪文本不应被索引: %v", seen)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func containsString(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

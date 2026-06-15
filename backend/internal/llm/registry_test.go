package llm

import "testing"

func TestRegistry_Resolve(t *testing.T) {
	p1 := NewMockProvider()
	p2 := NewMockProvider()
	reg := NewRegistry([]ModelProfile{
		{Key: "ds", Label: "DeepSeek", Provider: p1, Model: "deepseek-chat"},
		{Key: "gpt", Label: "GPT-4o", Provider: p2, Model: "gpt-4o"},
	}, "ds")

	// 空 key → 默认 profile（ds）
	if pv, m := reg.Resolve(""); pv != p1 || m != "deepseek-chat" {
		t.Errorf("空 key 应解析到默认；得到 (%p,%q)", pv, m)
	}
	// 命中 key → 该 profile
	if pv, m := reg.Resolve("gpt"); pv != p2 || m != "gpt-4o" {
		t.Errorf("key=gpt 应解析到 GPT profile；得到 (%p,%q)", pv, m)
	}
	// 未命中但非空 → 默认 provider + 把 key 当原始模型名（兼容 L1 raw-model 覆盖）
	if pv, m := reg.Resolve("deepseek-reasoner"); pv != p1 || m != "deepseek-reasoner" {
		t.Errorf("未命中 key 应回退默认 provider + 原始模型；得到 (%p,%q)", pv, m)
	}
}

func TestRegistry_OptionsAndHas(t *testing.T) {
	p := NewMockProvider()
	reg := NewRegistry([]ModelProfile{
		{Key: "a", Label: "Alpha", Provider: p, Model: "ma"},
		{Key: "b", Provider: p, Model: "mb"}, // 无 label → Options 回退用 key
	}, "a")

	opts := reg.Options()
	if len(opts) != 2 || opts[0].Key != "a" || opts[0].Label != "Alpha" || opts[1].Label != "b" {
		t.Errorf("Options 顺序/label 错: %+v", opts)
	}
	if !reg.Has("a") || reg.Has("zzz") {
		t.Errorf("Has 判断错: Has(a)=%v Has(zzz)=%v", reg.Has("a"), reg.Has("zzz"))
	}
	if reg.DefaultKey() != "a" {
		t.Errorf("DefaultKey=%q want a", reg.DefaultKey())
	}
}

func TestRegistry_DefaultKeyFallsBackToFirst(t *testing.T) {
	p := NewMockProvider()
	reg := NewRegistry([]ModelProfile{{Key: "x", Provider: p, Model: "mx"}}, "nonexistent")
	if reg.DefaultKey() != "x" {
		t.Errorf("不存在的 defaultKey 应回退到第一个；得到 %q", reg.DefaultKey())
	}
	if pv, m := reg.Default(); pv != p || m != "mx" {
		t.Errorf("Default() 错: (%p,%q)", pv, m)
	}
}

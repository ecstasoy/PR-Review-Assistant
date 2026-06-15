package llm

// ModelProfile 注册表里一个具名模型：一个 provider 端点 + 该端点下的模型名。
// 不同 profile 可指向不同 base_url + key（跨供应商），由构造方决定。
type ModelProfile struct {
	Key      string
	Label    string
	Provider Provider
	Model    string
}

// ModelOption 暴露给前端的可选模型（L3 白名单）；不含 provider / 端点等内部信息。
type ModelOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Registry 具名模型注册表：按 key 解析到 (Provider, model)。
// 默认项用于「未指定」与「未知 key」两种回退；保留声明顺序供前端列表。
type Registry struct {
	defaultKey string
	profiles   map[string]ModelProfile
	order      []string
}

// NewRegistry 构造注册表。defaultKey 不在 profiles 里时回退到第一个 profile 的 key。
func NewRegistry(profiles []ModelProfile, defaultKey string) *Registry {
	m := make(map[string]ModelProfile, len(profiles))
	order := make([]string, 0, len(profiles))
	for _, p := range profiles {
		m[p.Key] = p
		order = append(order, p.Key)
	}
	if _, ok := m[defaultKey]; !ok && len(order) > 0 {
		defaultKey = order[0]
	}
	return &Registry{defaultKey: defaultKey, profiles: m, order: order}
}

// Resolve key → (Provider, model)：
//   - 空 key → 默认 profile
//   - 命中 profile → 该 profile 的 provider + model
//   - 非空但未命中 → 默认 provider + 把 key 当原始模型名（兼容 L1 的 raw-model 按阶段覆盖）
func (r *Registry) Resolve(key string) (Provider, string) {
	def := r.profiles[r.defaultKey]
	if key == "" {
		return def.Provider, def.Model
	}
	if p, ok := r.profiles[key]; ok {
		return p.Provider, p.Model
	}
	return def.Provider, key
}

// Default 返回默认 profile 的 (Provider, model)。
func (r *Registry) Default() (Provider, string) { return r.Resolve("") }

// DefaultKey 默认 profile 的 key。
func (r *Registry) DefaultKey() string { return r.defaultKey }

// Has 报告 key 是否是已注册 profile（L3 白名单校验用）。
func (r *Registry) Has(key string) bool {
	_, ok := r.profiles[key]
	return ok
}

// Options 按声明顺序返回 {key,label}；label 空时回退到 key。
func (r *Registry) Options() []ModelOption {
	out := make([]ModelOption, 0, len(r.order))
	for _, k := range r.order {
		p := r.profiles[k]
		label := p.Label
		if label == "" {
			label = p.Key
		}
		out = append(out, ModelOption{Key: k, Label: label})
	}
	return out
}

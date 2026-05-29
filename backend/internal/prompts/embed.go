// Package prompts 把 *.tmpl 通过 go:embed 编进二进制，对外暴露按名拿模板。
package prompts

import (
	"embed"
	"fmt"
	"io/fs"
	"text/template"
)

//go:embed *.tmpl
var files embed.FS

// Parse 读取并编译指定模板。
func Parse(name string) (*template.Template, error) {
	src, err := fs.ReadFile(files, name)
	if err != nil {
		return nil, fmt.Errorf("prompt %q: %w", name, err)
	}
	return template.New(name).Parse(string(src))
}

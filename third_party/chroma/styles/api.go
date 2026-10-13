package styles

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
)

//go:embed *.xml
var embedded embed.FS

var (
	Registry     = map[string]*chroma.Style{}
	Fallback     *chroma.Style
	registryOnce sync.Once
)

func registry() map[string]*chroma.Style {
	registryOnce.Do(func() {
		files, err := fs.ReadDir(embedded, ".")
		if err != nil {
			panic(err)
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			r, err := embedded.Open(file.Name())
			if err != nil {
				panic(err)
			}
			style, err := chroma.NewXMLStyle(r)
			if err != nil {
				panic(err)
			}
			Registry[strings.ToLower(style.Name)] = style
			_ = r.Close()
		}
		Fallback = Registry["swapoff"]
	})
	return Registry
}

func Register(style *chroma.Style) *chroma.Style {
	registry()[strings.ToLower(style.Name)] = style
	return style
}

func Names() []string {
	out := []string{}
	for name := range registry() {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func Get(name string) *chroma.Style {
	if style, ok := registry()[strings.ToLower(name)]; ok {
		return style
	}
	return Fallback
}

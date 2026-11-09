package viz

import (
	"bytes"
	"strings"
	"testing"

	"github.com/katalystng/okf-go/bundle"
)

func TestRenderDrawsLinksAndSources(t *testing.T) {
	b, err := bundle.Load("../testdata/bundle")
	if err != nil {
		t.Fatal(err)
	}
	g := build(b, Options{Layers: []string{"spec"}})
	var edges []string
	for _, e := range g.Edges {
		edges = append(edges, e.Data["id"].(string)+"/"+e.Data["kind"].(string))
	}
	want := "product/engine__spec/sessions/link spec/sessions__intent/link"
	if got := strings.Join(edges, " "); got != want {
		t.Errorf("edges = %s, want %s", got, want)
	}
	if g.Layers[0].Name != "spec" || g.Layers[0].Color != palette[0] {
		t.Errorf("layers = %v, want spec first", g.Layers)
	}
}

func TestRenderIsStable(t *testing.T) {
	b, err := bundle.Load("../testdata/bundle")
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := Render(b, Options{Name: "t"})
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		again, _, _ := Render(b, Options{Name: "t"})
		if !bytes.Equal(first, again) {
			t.Fatal("two renders of one bundle differ")
		}
	}
	if bytes.Contains(first, []byte("__BUNDLE_DATA__")) || bytes.Contains(first, []byte("/*__VIZ_JS__*/")) {
		t.Error("a placeholder survived the render")
	}
}

func TestDefaultName(t *testing.T) {
	for dir, want := range map[string]string{"/r/ssh-mcp/docs": "ssh-mcp", "/r/kb": "kb", "/r/kb/": "kb"} {
		if got := DefaultName(dir); got != want {
			t.Errorf("DefaultName(%q) = %q, want %q", dir, got, want)
		}
	}
}

// Package viz renders an OKF bundle as one self-contained HTML page: a graph of its concepts with a
// detail pane for each. The page is a port of the viewer in GoogleCloudPlatform/knowledge-catalog.
//
// The output depends on the bundle alone, so a committed page changes only when the bundle does.
// Staleness, which depends on the time of viewing, is computed in the page.
package viz

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/katalystng/okf-go/bundle"
)

var (
	//go:embed assets/viz.html
	templateHTML string
	//go:embed assets/viz.css
	css string
	//go:embed assets/viz.js
	js string
)

// palette colors the layers in order. A bundle with more layers than colors reuses them.
var palette = []string{
	"#3b82f6", "#10b981", "#f59e0b", "#8b5cf6", "#ef4444",
	"#06b6d4", "#84cc16", "#ec4899", "#64748b", "#a16207",
}

// RootLayer names the layer of a concept at the bundle root.
const RootLayer = "(root)"

// Options adjusts the page.
type Options struct {
	// Name is the bundle's display name. It defaults to the bundle directory's name.
	Name string
	// Layers orders the layers, by top-level folder name, and so sets their colors. A folder left
	// out follows the listed ones in name order. Empty orders every layer by name.
	Layers []string
}

// Stats counts what a render drew.
type Stats struct {
	Concepts int
	Edges    int
	Bytes    int
}

type element struct {
	Data map[string]any `json:"data"`
}

type layer struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type graph struct {
	Nodes  []element         `json:"nodes"`
	Edges  []element         `json:"edges"`
	Bodies map[string]string `json:"bodies"`
	Types  []string          `json:"types"`
	Layers []layer           `json:"layers"`
}

// Render writes the bundle's page.
func Render(b *bundle.Bundle, opts Options) ([]byte, Stats, error) {
	g := build(b, opts)
	data, err := marshal(g)
	if err != nil {
		return nil, Stats{}, err
	}
	name, err := marshal(opts.Name)
	if err != nil {
		return nil, Stats{}, err
	}
	page := strings.NewReplacer(
		"/*__VIZ_CSS__*/", css,
		"/*__VIZ_JS__*/", js,
		"__BUNDLE_NAME__", string(name),
		"__BUNDLE_DATA__", string(data),
	).Replace(templateHTML)
	return []byte(page), Stats{Concepts: len(g.Nodes), Edges: len(g.Edges), Bytes: len(page)}, nil
}

func build(b *bundle.Bundle, opts Options) graph {
	g := graph{Bodies: map[string]string{}, Nodes: []element{}, Edges: []element{}}
	colors := layerColors(b, opts.Layers, &g)

	types := map[string]bool{}
	for _, c := range b.Concepts {
		typ := c.String("type")
		if typ == "" {
			typ = "Unknown"
		}
		types[typ] = true
		label := c.String("title")
		if label == "" {
			label = c.ID
		}
		l := layerOf(c.ID)
		d := map[string]any{
			"id":          c.ID,
			"label":       label,
			"type":        typ,
			"layer":       l,
			"color":       colors[l],
			"description": c.String("description"),
			"resource":    c.String("resource"),
			"tags":        nonNil(c.Strings("tags")),
			"status":      c.Status(),
			"generated":   c.Frontmatter["generated"],
			"verified":    nonNilMaps(c.Maps("verified")),
			"stale_after": c.String("stale_after"),
			"sources":     nonNilMaps(c.Maps("sources")),
			"trust_tier":  c.TrustTier(),
			"size":        30 + min(60, len(c.Body)/200),
		}
		if a, ok := c.Frontmatter["attribution"].(map[string]any); ok {
			d["attribution"] = a
		}
		g.Nodes = append(g.Nodes, element{Data: d})
		g.Bodies[c.ID] = c.Body
	}
	for t := range types {
		g.Types = append(g.Types, t)
	}
	slices.Sort(g.Types)

	seen := map[string]bool{}
	addEdge := func(from, to, kind string) {
		if to == from || b.Concept(to) == nil {
			return
		}
		id := from + "__" + to
		if seen[id] {
			return
		}
		seen[id] = true
		g.Edges = append(g.Edges, element{Data: map[string]any{"id": id, "source": from, "target": to, "kind": kind}})
	}
	for _, c := range b.Concepts {
		for _, l := range c.Links {
			if l.External() {
				continue
			}
			if target := bundle.Resolve(c.Path, l.Target); strings.HasSuffix(target, ".md") {
				addEdge(c.ID, strings.TrimSuffix(target, ".md"), "link")
			}
		}
		for _, s := range c.Maps("sources") {
			res, _ := s["resource"].(string)
			if strings.HasPrefix(res, "/") && strings.HasSuffix(res, ".md") {
				addEdge(c.ID, strings.TrimSuffix(bundle.Resolve(c.Path, res), ".md"), "source")
			}
		}
	}
	return g
}

// layerOf returns a concept's layer: its top-level folder, or RootLayer.
func layerOf(id string) string {
	if dir, _, ok := strings.Cut(id, "/"); ok {
		return dir
	}
	return RootLayer
}

// layerColors orders the bundle's layers, records them in g, and returns each one's color.
func layerColors(b *bundle.Bundle, order []string, g *graph) map[string]string {
	present := map[string]bool{}
	for _, c := range b.Concepts {
		present[layerOf(c.ID)] = true
	}
	var names []string
	for _, n := range order {
		if present[n] && !slices.Contains(names, n) {
			names = append(names, n)
		}
	}
	var rest []string
	for n := range present {
		if !slices.Contains(names, n) {
			rest = append(rest, n)
		}
	}
	slices.Sort(rest)
	names = append(names, rest...)

	colors := map[string]string{}
	for i, n := range names {
		colors[n] = palette[i%len(palette)]
		g.Layers = append(g.Layers, layer{Name: n, Color: colors[n]})
	}
	return colors
}

// marshal encodes v as JSON safe to place inside a script element.
func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("encode the bundle: %w", err)
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func nonNilMaps(s []map[string]any) []map[string]any {
	if s == nil {
		return []map[string]any{}
	}
	return s
}

// DefaultName returns the display name for a bundle at dir: the name of the folder, or of its
// parent when the folder is named docs.
func DefaultName(dir string) string {
	base := path.Base(strings.TrimRight(dir, "/"))
	if base == "docs" {
		if parent := path.Base(path.Dir(strings.TrimRight(dir, "/"))); parent != "." && parent != "/" {
			return parent
		}
	}
	return base
}

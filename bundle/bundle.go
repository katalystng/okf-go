// Package bundle reads an Open Knowledge Format (OKF) v0.2 bundle: its concept documents, its index
// files, and its log files, with the links between them.
package bundle

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// IndexName and LogName are the reserved filenames of OKF v0.2 section 3.1.
const (
	IndexName = "index.md"
	LogName   = "log.md"
)

// Bundle is a bundle read from disk. Paths in it are bundle-relative and slash-separated.
type Bundle struct {
	// Root is the bundle's directory on disk.
	Root string
	// Concepts holds each concept document, sorted by ID.
	Concepts []*Concept
	// Indexes holds each index.md, sorted by path.
	Indexes []*Document
	// Logs holds each log.md, sorted by path.
	Logs []*Document
	// Invalid holds each concept document whose frontmatter failed to parse, sorted by path.
	Invalid []*Document
}

// Document is one markdown file of a bundle.
type Document struct {
	// Path is the file's bundle-relative path, such as spec/claims.md.
	Path string
	// Frontmatter is the parsed frontmatter, or nil when the file has none.
	Frontmatter map[string]any
	// HasFrontmatter reports whether the file opens with a frontmatter block.
	HasFrontmatter bool
	// Body is the markdown after the frontmatter.
	Body string
	// BodyLine is the 0-based file line the body starts on.
	BodyLine int
	// Links holds each markdown link in the body, in order.
	Links []Link
	// Err holds the reason the frontmatter failed to parse.
	Err error
}

// Concept is a concept document: any markdown file other than a reserved filename.
type Concept struct {
	Document
	// ID is the path without its .md suffix, such as spec/claims.
	ID string
}

// Link is a markdown link in a document's body.
type Link struct {
	// Text is the link's text.
	Text string
	// Target is the link's destination as written, fragment included.
	Target string
	// Line is the 1-based line of the link in the file.
	Line int
}

// External reports whether the link names a URL with a scheme, or a fragment alone.
func (l Link) External() bool {
	return strings.HasPrefix(l.Target, "#") || schemeRE.MatchString(l.Target)
}

var (
	// linkRE matches an inline markdown link and captures its text and destination.
	linkRE   = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	schemeRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
	fenceRE  = regexp.MustCompile("^\\s*(```|~~~)")
)

// Load reads the bundle rooted at root. A file whose frontmatter fails to parse lands in Invalid;
// an I/O failure returns an error.
func Load(root string) (*Bundle, error) {
	b := &Bundle{Root: root}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		doc := Parse(filepath.ToSlash(rel), raw)
		switch path.Base(doc.Path) {
		case IndexName:
			b.Indexes = append(b.Indexes, doc)
		case LogName:
			b.Logs = append(b.Logs, doc)
		default:
			if doc.Err != nil {
				b.Invalid = append(b.Invalid, doc)
				return nil
			}
			b.Concepts = append(b.Concepts, &Concept{Document: *doc, ID: strings.TrimSuffix(doc.Path, ".md")})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read bundle %s: %w", root, err)
	}
	slices.SortFunc(b.Concepts, func(a, c *Concept) int { return strings.Compare(a.ID, c.ID) })
	for _, docs := range [][]*Document{b.Indexes, b.Logs, b.Invalid} {
		slices.SortFunc(docs, func(a, c *Document) int { return strings.Compare(a.Path, c.Path) })
	}
	return b, nil
}

// Parse splits a markdown file into its frontmatter and body, per OKF v0.2 section 4. A file whose
// first line is --- opens a frontmatter block that ends at the next --- line.
func Parse(relPath string, raw []byte) *Document {
	doc := &Document{Path: relPath}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	bodyStart := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		doc.HasFrontmatter = true
		end := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				end = i
				break
			}
		}
		if end < 0 {
			doc.Err = errors.New("the frontmatter block has no closing --- line")
			doc.Body = text
			return doc
		}
		doc.Frontmatter, doc.Err = decodeFrontmatter(strings.Join(lines[1:end], "\n"))
		bodyStart = end + 1
	}
	doc.Body = strings.Join(lines[bodyStart:], "\n")
	doc.BodyLine = bodyStart
	doc.Links = extractLinks(lines, bodyStart)
	return doc
}

// decodeFrontmatter parses a frontmatter block as a YAML mapping. It decodes through yaml.Node, so a
// timestamp stays the text the author wrote, and a repeated key is an error.
func decodeFrontmatter(src string) (map[string]any, error) {
	var root yaml.Node
	dec := yaml.NewDecoder(bytes.NewBufferString(src))
	if err := dec.Decode(&root); err != nil {
		if errors.Is(err, io.EOF) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("the frontmatter is invalid YAML: %w", err)
	}
	if len(root.Content) == 0 {
		return map[string]any{}, nil
	}
	node := root.Content[0]
	if node.Kind != yaml.MappingNode {
		return nil, errors.New("the frontmatter is not a YAML mapping")
	}
	value, err := nodeValue(node)
	if err != nil {
		return nil, err
	}
	return value.(map[string]any), nil
}

// nodeValue converts a YAML node into plain Go values, keeping each scalar other than a bool, an
// int, a float, or null as its source text.
func nodeValue(n *yaml.Node) (any, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return nodeValue(n.Content[0])
	case yaml.AliasNode:
		return nodeValue(n.Alias)
	case yaml.MappingNode:
		m := make(map[string]any, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i].Value
			if _, dup := m[key]; dup {
				return nil, fmt.Errorf("the frontmatter repeats the key %q on line %d", key, n.Content[i].Line+1)
			}
			v, err := nodeValue(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			m[key] = v
		}
		return m, nil
	case yaml.SequenceNode:
		s := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := nodeValue(c)
			if err != nil {
				return nil, err
			}
			s = append(s, v)
		}
		return s, nil
	default:
		switch n.Tag {
		case "!!bool", "!!int", "!!float", "!!null":
			var v any
			if err := n.Decode(&v); err != nil {
				return nil, err
			}
			return v, nil
		}
		return n.Value, nil
	}
}

// extractLinks finds the inline markdown links in the body, skipping fenced code blocks and inline
// code spans.
func extractLinks(lines []string, bodyStart int) []Link {
	var links []Link
	inFence := false
	for i := bodyStart; i < len(lines); i++ {
		line := lines[i]
		if fenceRE.MatchString(line) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, m := range linkRE.FindAllStringSubmatchIndex(line, -1) {
			if insideCodeSpan(line, m[0]) {
				continue
			}
			links = append(links, Link{Text: line[m[2]:m[3]], Target: line[m[4]:m[5]], Line: i + 1})
		}
	}
	return links
}

// insideCodeSpan reports whether position pos of line falls inside a backtick code span.
func insideCodeSpan(line string, pos int) bool {
	return strings.Count(line[:pos], "`")%2 == 1
}

// Resolve returns the bundle-relative path a link from the document at docPath points at, with its
// fragment removed. A target that starts with / is bundle-relative, per OKF v0.2 section 6.1. The
// result can lie outside the bundle, as a path starting with ../.
func Resolve(docPath, target string) string {
	target, _, _ = strings.Cut(target, "#")
	if target == "" {
		return docPath
	}
	if strings.HasPrefix(target, "/") {
		return path.Clean(strings.TrimPrefix(target, "/"))
	}
	return path.Clean(path.Join(path.Dir(docPath), target))
}

// Concept returns the concept with the given ID, or nil.
func (b *Bundle) Concept(id string) *Concept {
	i, ok := slices.BinarySearchFunc(b.Concepts, id, func(c *Concept, id string) int { return strings.Compare(c.ID, id) })
	if !ok {
		return nil
	}
	return b.Concepts[i]
}

// String returns a frontmatter value as a string, or "" when it is absent or not a scalar.
func (d *Document) String(key string) string {
	switch v := d.Frontmatter[key].(type) {
	case string:
		return v
	case nil:
		return ""
	case map[string]any, []any:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

// Strings returns a frontmatter list of scalars, or a single scalar as a one-element list.
func (d *Document) Strings(key string) []string {
	switch v := d.Frontmatter[key].(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			switch e.(type) {
			case map[string]any, []any, nil:
				continue
			}
			out = append(out, fmt.Sprint(e))
		}
		return out
	case nil:
		return nil
	case map[string]any:
		return nil
	default:
		return []string{fmt.Sprint(v)}
	}
}

// Maps returns a frontmatter value as a list of mappings. A single mapping counts as a one-element
// list, as OKF v0.2 section 5.2 directs for verified.
func (d *Document) Maps(key string) []map[string]any {
	switch v := d.Frontmatter[key].(type) {
	case map[string]any:
		return []map[string]any{v}
	case []any:
		var out []map[string]any
		for _, e := range v {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

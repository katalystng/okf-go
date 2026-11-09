// Package check checks an OKF v0.2 bundle. A conformance failure, per OKF v0.2 section 11, is an
// error. A departure from the guidance the specification gives as SHOULD is a warning: a link that
// resolves to nothing, an index that leaves a file out, or a malformed trust or lifecycle field.
// Fields a bundle adds of its own, such as attribution, are checked when they are present.
package check

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/katalystng/okf-go/bundle"
)

// Severity orders findings: an error fails conformance, and a warning departs from guidance.
type Severity int

const (
	Warning Severity = iota
	Error
)

func (s Severity) String() string {
	if s == Error {
		return "error"
	}
	return "warning"
}

// Finding is one problem in a bundle.
type Finding struct {
	Severity Severity
	// Path is the bundle-relative file the finding is about.
	Path string
	// Line is the 1-based line, or 0 for the file as a whole.
	Line    int
	Message string
}

func (f Finding) String() string {
	loc := f.Path
	if f.Line > 0 {
		loc = fmt.Sprintf("%s:%d", f.Path, f.Line)
	}
	return fmt.Sprintf("%s: %s: %s", loc, f.Severity, f.Message)
}

var (
	statuses    = []string{"draft", "stable", "deprecated"}
	derivations = []string{"original", "quoted", "paraphrased", "inferred", "revised"}
	logDateRE   = regexp.MustCompile(`^##\s+(.*)$`)
	isoDateRE   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	indexItemRE = regexp.MustCompile(`^\s*[*-]\s+\[[^\]]*\]\(([^)\s]+)\)\s*(?:-\s*(.*))?$`)
)

// Run checks the bundle and returns its findings, sorted by path and line.
func Run(b *bundle.Bundle) []Finding {
	var fs []Finding
	add := func(sev Severity, p string, line int, format string, args ...any) {
		fs = append(fs, Finding{Severity: sev, Path: p, Line: line, Message: fmt.Sprintf(format, args...)})
	}

	for _, d := range b.Invalid {
		add(Error, d.Path, 0, "%v", d.Err)
	}
	for _, c := range b.Concepts {
		checkConcept(b, c, add)
	}
	for _, d := range b.Indexes {
		checkIndex(b, d, add)
	}
	for _, d := range b.Logs {
		checkLog(d, add)
	}
	for _, d := range allDocs(b) {
		checkLinks(b, d, add)
	}

	slices.SortStableFunc(fs, func(a, c Finding) int {
		if a.Path != c.Path {
			return strings.Compare(a.Path, c.Path)
		}
		return a.Line - c.Line
	})
	return fs
}

type addFunc func(sev Severity, p string, line int, format string, args ...any)

func checkConcept(b *bundle.Bundle, c *bundle.Concept, add addFunc) {
	if !c.HasFrontmatter {
		add(Error, c.Path, 0, "a concept document opens with a frontmatter block")
		return
	}
	if strings.TrimSpace(c.String("type")) == "" {
		add(Error, c.Path, 0, "the frontmatter has no type")
	}
	if s := c.String("status"); s != "" && !slices.Contains(statuses, s) {
		add(Warning, c.Path, 0, "status %q is none of %s", s, strings.Join(statuses, ", "))
	}
	if s := c.String("stale_after"); s != "" {
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			add(Warning, c.Path, 0, "stale_after %q is no ISO 8601 datetime with an offset", s)
		}
	}
	if g, ok := c.Frontmatter["generated"]; ok {
		m, isMap := g.(map[string]any)
		if !isMap || fmt.Sprint(m["by"]) == "" || m["by"] == nil {
			add(Warning, c.Path, 0, "generated holds no by")
		}
	}
	if _, ok := c.Frontmatter["verified"]; ok {
		events := c.Maps("verified")
		if len(events) == 0 {
			add(Warning, c.Path, 0, "verified holds no { by, at } event")
		}
		for _, e := range events {
			if e["by"] == nil || e["at"] == nil {
				add(Warning, c.Path, 0, "a verified event lacks by or at")
			}
		}
	}
	for i, s := range c.Maps("sources") {
		if s["resource"] == nil {
			add(Warning, c.Path, 0, "source %d has no resource", i+1)
			continue
		}
		res := fmt.Sprint(s["resource"])
		if strings.HasPrefix(res, "/") && !exists(b, bundle.Resolve(c.Path, res)) {
			add(Warning, c.Path, 0, "source %q names no file in the bundle", res)
		}
	}
	if a, ok := c.Frontmatter["attribution"]; ok {
		checkAttribution(c, a, add)
	}
}

// checkAttribution checks the attribution extension: a derivation, the authors whose words the text
// is, and the contributors whose ideas it draws on.
func checkAttribution(c *bundle.Concept, a any, add addFunc) {
	m, ok := a.(map[string]any)
	if !ok {
		add(Warning, c.Path, 0, "attribution is not a mapping")
		return
	}
	d := fmt.Sprint(m["derivation"])
	if !slices.Contains(derivations, d) {
		add(Warning, c.Path, 0, "attribution.derivation %q is none of %s", d, strings.Join(derivations, ", "))
	}
	if authors, ok := m["authors"].([]any); !ok || len(authors) == 0 {
		add(Warning, c.Path, 0, "attribution.authors lists no author")
	}
	if v, ok := m["contributors"]; ok {
		if _, ok := v.([]any); !ok {
			add(Warning, c.Path, 0, "attribution.contributors is not a list")
		}
	}
}

// checkIndex checks an index.md: frontmatter only at the bundle root with okf_version alone, and an
// entry for each file and subdirectory beside it, per OKF v0.2 section 8.
func checkIndex(b *bundle.Bundle, d *bundle.Document, add addFunc) {
	dir := path.Dir(d.Path)
	if d.Err != nil {
		add(Error, d.Path, 0, "%v", d.Err)
	}
	if d.HasFrontmatter {
		if dir != "." {
			add(Error, d.Path, 0, "an index.md below the bundle root holds no frontmatter")
		} else {
			for k := range d.Frontmatter {
				if k != "okf_version" {
					add(Error, d.Path, 0, "the root index.md frontmatter holds okf_version alone, and it holds %q", k)
				}
			}
		}
	}

	listed := map[string]string{}
	for i, line := range strings.Split(d.Body, "\n") {
		m := indexItemRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		target := bundle.Resolve(d.Path, m[1])
		listed[strings.TrimSuffix(target, "/")] = strings.TrimSpace(m[2])
		if c := b.Concept(strings.TrimSuffix(target, ".md")); c != nil && m[2] != "" {
			if desc := c.String("description"); desc != "" && strings.TrimSpace(m[2]) != desc {
				add(Warning, d.Path, lineOf(d, i), "the entry for %s differs from its description", c.Path)
			}
		}
	}

	entries, err := os.ReadDir(filepath.Join(b.Root, filepath.FromSlash(dir)))
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == bundle.IndexName || name == bundle.LogName {
			continue
		}
		rel := path.Join(dir, name)
		if e.IsDir() {
			if !hasMarkdown(filepath.Join(b.Root, filepath.FromSlash(rel))) {
				continue
			}
			if _, ok := listed[rel]; !ok {
				if _, ok := listed[path.Join(rel, bundle.IndexName)]; !ok {
					add(Warning, d.Path, 0, "the index leaves out the folder %s/", name)
				}
			}
			continue
		}
		if path.Ext(name) == ".md" {
			if _, ok := listed[rel]; !ok {
				add(Warning, d.Path, 0, "the index leaves out %s", name)
			}
		}
	}
}

// checkLog checks that each date heading of a log.md is YYYY-MM-DD, per OKF v0.2 section 9.
func checkLog(d *bundle.Document, add addFunc) {
	for i, line := range strings.Split(d.Body, "\n") {
		if m := logDateRE.FindStringSubmatch(line); m != nil && !isoDateRE.MatchString(strings.TrimSpace(m[1])) {
			add(Error, d.Path, lineOf(d, i), "the log heading %q is no YYYY-MM-DD date", strings.TrimSpace(m[1]))
		}
	}
}

// checkLinks reports each relative or bundle-relative link whose file is absent. OKF v0.2 section
// 6.1 lets consumers tolerate a broken link, so it is a warning.
func checkLinks(b *bundle.Bundle, d *bundle.Document, add addFunc) {
	for _, l := range d.Links {
		if l.External() {
			continue
		}
		target := bundle.Resolve(d.Path, l.Target)
		p := filepath.Join(b.Root, filepath.FromSlash(target))
		if _, err := os.Stat(p); err != nil {
			add(Warning, d.Path, l.Line, "the link to %s resolves to no file", l.Target)
		}
	}
}

func allDocs(b *bundle.Bundle) []*bundle.Document {
	var docs []*bundle.Document
	for _, c := range b.Concepts {
		docs = append(docs, &c.Document)
	}
	docs = append(docs, b.Indexes...)
	return append(docs, b.Logs...)
}

func exists(b *bundle.Bundle, rel string) bool {
	_, err := os.Stat(filepath.Join(b.Root, filepath.FromSlash(rel)))
	return err == nil
}

func hasMarkdown(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(p) == ".md" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// lineOf turns a 0-based body line into a 1-based file line.
func lineOf(d *bundle.Document, bodyLine int) int {
	return d.BodyLine + bodyLine + 1
}

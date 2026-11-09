package bundle

import (
	"slices"
	"testing"
)

func TestLoadSortsTheBundle(t *testing.T) {
	b, err := Load("../testdata/bundle")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, c := range b.Concepts {
		ids = append(ids, c.ID)
	}
	want := []string{"intent", "product/engine", "product/orphan", "spec/sessions"}
	if !slices.Equal(ids, want) {
		t.Errorf("concepts = %v, want %v", ids, want)
	}
	if len(b.Indexes) != 3 {
		t.Errorf("indexes = %d, want 3", len(b.Indexes))
	}
}

func TestParseKeepsTimestampsAsWritten(t *testing.T) {
	d := Parse("a.md", []byte("---\ntype: T\nstale_after: 2026-01-01T00:00:00Z\ncount: 3\n---\nbody\n"))
	if d.Err != nil {
		t.Fatal(d.Err)
	}
	if got := d.Frontmatter["stale_after"]; got != "2026-01-01T00:00:00Z" {
		t.Errorf("stale_after = %#v", got)
	}
	if got := d.Frontmatter["count"]; got != 3 {
		t.Errorf("count = %#v, want the int 3", got)
	}
	if d.BodyLine != 5 || d.Body != "body\n" {
		t.Errorf("body = %q at line %d", d.Body, d.BodyLine)
	}
}

func TestParseRejectsARepeatedKey(t *testing.T) {
	d := Parse("a.md", []byte("---\ntype: A\ntype: B\n---\n"))
	if d.Err == nil {
		t.Fatal("a repeated key parsed")
	}
}

func TestParseRejectsAnUnclosedBlock(t *testing.T) {
	if d := Parse("a.md", []byte("---\ntype: A\n")); d.Err == nil {
		t.Fatal("an unclosed block parsed")
	}
}

func TestLinksSkipCode(t *testing.T) {
	b, err := Load("../testdata/bundle")
	if err != nil {
		t.Fatal(err)
	}
	var targets []string
	for _, l := range b.Concept("spec/sessions").Links {
		targets = append(targets, l.Target)
	}
	want := []string{"../intent.md", "/intent.md#top", "missing.md", "https://example.com"}
	if !slices.Equal(targets, want) {
		t.Errorf("links = %v, want %v", targets, want)
	}
}

func TestResolve(t *testing.T) {
	for _, tc := range []struct{ doc, target, want string }{
		{"spec/a.md", "../intent.md", "intent.md"},
		{"spec/a.md", "/intent.md#x", "intent.md"},
		{"spec/a.md", "b.md", "spec/b.md"},
		{"spec/a.md", "#x", "spec/a.md"},
	} {
		if got := Resolve(tc.doc, tc.target); got != tc.want {
			t.Errorf("Resolve(%q, %q) = %q, want %q", tc.doc, tc.target, got, tc.want)
		}
	}
}

func TestTrustTier(t *testing.T) {
	b, err := Load("../testdata/bundle")
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{
		"intent":         TierHumanReviewed,
		"product/engine": TierMachineConfirmed,
		"spec/sessions":  TierUnverified,
	} {
		if got := b.Concept(id).TrustTier(); got != want {
			t.Errorf("%s tier = %s, want %s", id, got, want)
		}
	}
}

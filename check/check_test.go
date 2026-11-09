package check

import (
	"slices"
	"testing"

	"github.com/katalystng/okf-go/bundle"
)

func TestRunReportsEachProblem(t *testing.T) {
	b, err := bundle.Load("../testdata/bundle")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range Run(b) {
		got = append(got, f.String())
	}
	want := []string{
		"product/index.md: warning: the index leaves out orphan.md",
		"product/index.md:3: warning: the entry for product/engine.md differs from its description",
		"product/orphan.md: error: the frontmatter has no type",
		"spec/sessions.md:16: warning: the link to missing.md resolves to no file",
	}
	if !slices.Equal(got, want) {
		t.Errorf("findings:\n%q\nwant:\n%q", got, want)
	}
}

func TestRunChecksOwnDocs(t *testing.T) {
	b, err := bundle.Load("../docs")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range Run(b) {
		t.Error(f)
	}
}

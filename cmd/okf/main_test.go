package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestVizCheckFailsUntilThePageIsWritten(t *testing.T) {
	out := filepath.Join(t.TempDir(), "viz.html")
	var buf bytes.Buffer
	args := []string{"viz", "-o", out, "-name", "t", "../../testdata/bundle"}
	if err := run(append([]string{"viz", "-check"}, args[1:]...), &buf, &buf); !errors.Is(err, errFailed) {
		t.Fatalf("check before write = %v, want a failure", err)
	}
	if err := run(args, &buf, &buf); err != nil {
		t.Fatal(err)
	}
	if err := run(append([]string{"viz", "-check"}, args[1:]...), &buf, &buf); err != nil {
		t.Fatalf("check after write = %v", err)
	}
	if err := os.WriteFile(out, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(append([]string{"viz", "-check"}, args[1:]...), &buf, &buf); !errors.Is(err, errFailed) {
		t.Fatalf("check of a stale page = %v, want a failure", err)
	}
}

func TestCheckFailsOnAnError(t *testing.T) {
	var buf bytes.Buffer
	if err := run([]string{"check", "../../testdata/bundle"}, &buf, &buf); !errors.Is(err, errFailed) {
		t.Fatalf("check = %v, want a failure for the untyped concept", err)
	}
}

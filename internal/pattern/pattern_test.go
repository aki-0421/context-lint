package pattern

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandDirectoryIncludesMarkdownAndAppliesExcludes(t *testing.T) {
	root := t.TempDir()
	writePatternFile(t, root, "docs/index.md")
	writePatternFile(t, root, "docs/secret.md")
	writePatternFile(t, root, "docs/image.png")

	excludes := NewMatcher(root, []string{"docs/secret.md"})
	got, empty, err := Expand(root, []string{"docs"}, excludes, true)
	if err != nil {
		t.Fatalf("Expand returned error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty = %#v, want none", empty)
	}
	if len(got) != 1 || got[0] != "docs/index.md" {
		t.Fatalf("targets = %#v, want docs/index.md only", got)
	}
}

func TestMatcherSupportsDoublestar(t *testing.T) {
	m := NewMatcher(t.TempDir(), []string{"docs/**/*.md"})
	if !m.Match("docs/spec/context-lint.md") {
		t.Fatal("matcher should match nested Markdown path")
	}
	if m.Match("README.md") {
		t.Fatal("matcher should not match README.md")
	}
}

func writePatternFile(t *testing.T, root string, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

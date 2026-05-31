package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverUsesDocumentedOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".context-lint.json"), `{"linter":{"document":{"entry":"JSON.md"}}}`)
	writeFile(t, filepath.Join(dir, ".context-lint.yaml"), "linter:\n  document:\n    entry: YAML.md\n")

	got, err := Discover(dir, "")
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if filepath.Base(got) != ".context-lint.yaml" {
		t.Fatalf("Discover() = %q, want .context-lint.yaml", got)
	}
}

func TestLoadJSONC(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".context-lint.jsonc")
	writeFile(t, configPath, `{
	  // This comment should be ignored.
	  "linter": {
	    "document": {
	      "entry": "AGENTS.md",
	      "requiredReachable": ["docs/**/*.md"],
	      "excludes": ["README.md"]
	    }
	  }
	}`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Linter.Document.Entry != "AGENTS.md" {
		t.Fatalf("entry = %q, want AGENTS.md", cfg.Linter.Document.Entry)
	}
	if got := cfg.Linter.Document.RequiredReachable[0]; got != "docs/**/*.md" {
		t.Fatalf("requiredReachable[0] = %q, want docs/**/*.md", got)
	}
}

func TestLoadFrontMatterExcludeFileNames(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".context-lint.yaml")
	writeFile(t, configPath, `linter:
  document:
    entry: AGENTS.md
    frontMatter:
      excludeFileNames:
        - README.md
        - docs/CHANGELOG.md
        - docs\ROUTES.md
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	got := cfg.Linter.Document.FrontMatter.ExcludeFileNames
	want := []string{"README.md", "CHANGELOG.md", "ROUTES.md"}
	if len(got) != len(want) {
		t.Fatalf("excludeFileNames = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("excludeFileNames = %#v, want %#v", got, want)
		}
	}
}

func TestLoadRequiresEntry(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".context-lint.yaml")
	writeFile(t, configPath, "linter:\n  document: {}\n")

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load returned nil error, want missing entry error")
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

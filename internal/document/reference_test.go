package document

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aki-0421/context-lint/internal/pattern"
)

func TestExtractReferencesFindsLinksHTMLAndPathText(t *testing.T) {
	data := []byte(`# Title

[Guide](docs/guide.md)
<a href="docs/api.md">API</a>
See docs/architecture/index.md for details.

` + "```" + `text
docs/generated.md
` + "```" + `
`)

	refs := ExtractReferences("AGENTS.md", data)
	assertHasRef(t, refs, "docs/guide.md", KindMarkdownLink)
	assertHasRef(t, refs, "docs/api.md", KindHTMLAttr)
	assertHasRef(t, refs, "docs/architecture/index.md", KindPathText)
	assertHasRef(t, refs, "docs/generated.md", KindPathCode)
}

func TestBuildGraphDetectsMissingReferencesAndReachability(t *testing.T) {
	root := t.TempDir()
	writeDocFile(t, root, "AGENTS.md", "[Docs](docs/index.md)\n")
	writeDocFile(t, root, "docs/index.md", "See missing.md and [API](api.md).\n")
	writeDocFile(t, root, "docs/api.md", "# API\n")

	graph, diags := BuildGraph(BuildOptions{
		Root:     root,
		Entry:    "AGENTS.md",
		Excludes: pattern.NewMatcher(root, nil),
	})

	if !graph.Reachable["docs/api.md"] {
		t.Fatal("docs/api.md should be reachable")
	}
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %d, want 1: %#v", len(diags), diags)
	}
	if diags[0].Code != "CL003" || diags[0].Reference != "missing.md" {
		t.Fatalf("diagnostic = %#v, want CL003 for missing.md", diags[0])
	}
}

func TestBuildGraphIgnoresExternalDomainLikePaths(t *testing.T) {
	root := t.TempDir()
	writeDocFile(t, root, "AGENTS.md", "Install github.com/aki-0421/context-lint/cmd/context-lint@latest\n")

	_, diags := BuildGraph(BuildOptions{
		Root:     root,
		Entry:    "AGENTS.md",
		Excludes: pattern.NewMatcher(root, nil),
	})
	if len(diags) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diags)
	}
}

func assertHasRef(t *testing.T, refs []Reference, raw string, kind string) {
	t.Helper()
	for _, ref := range refs {
		if ref.TargetRaw == raw && ref.Kind == kind {
			return
		}
	}
	t.Fatalf("missing reference raw=%q kind=%q in %#v", raw, kind, refs)
}

func writeDocFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

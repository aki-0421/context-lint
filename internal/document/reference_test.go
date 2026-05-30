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

func TestExtractReferencesIgnoresGoPackagePatterns(t *testing.T) {
	data := []byte(`Run ` + "`go test ./...`" + ` before opening a pull request.

` + "```" + `bash
go test ./...
go test ../...
` + "```" + `
`)

	refs := ExtractReferences("README.md", data)
	if len(refs) != 0 {
		t.Fatalf("refs = %#v, want no references", refs)
	}
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

func TestBuildGraphIgnoresMissingNonMarkdownReferences(t *testing.T) {
	root := t.TempDir()
	writeDocFile(t, root, "AGENTS.md", `[Docs](docs/index.md)
Missing non-doc files: components.json, ./ProjectListScreen.types, @/*, ./src/*.

`+"```tsx"+`
import type { ProjectListScreenProps } from "./ProjectListScreen.types";

export function ProjectListScreen() {
  return <main>{/* project list */}</main>;
}
`+"```"+`
`)
	writeDocFile(t, root, "docs/index.md", "# Docs\n")

	_, diags := BuildGraph(BuildOptions{
		Root:     root,
		Entry:    "AGENTS.md",
		Excludes: pattern.NewMatcher(root, nil),
	})
	if len(diags) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diags)
	}
}

func TestShouldReportMissingReferenceOnlyForMarkdownTargets(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{raw: "missing.md", want: true},
		{raw: "docs/*.md", want: true},
		{raw: "docs/guide.mdx#usage", want: true},
		{raw: "components.json", want: false},
		{raw: "./ProjectListScreen.types", want: false},
		{raw: "{/*", want: false},
		{raw: "*/}", want: false},
		{raw: "@/*", want: false},
		{raw: "./src/*", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := shouldReportMissingReference(tt.raw); got != tt.want {
				t.Fatalf("shouldReportMissingReference(%q) = %t, want %t", tt.raw, got, tt.want)
			}
		})
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

func TestBuildGraphDoesNotExpandPathLikeDirectories(t *testing.T) {
	root := t.TempDir()
	writeDocFile(t, root, "AGENTS.md", "```txt\napps/web/src\n```\nSee packages/ui.\n")
	writeDocFile(t, root, "apps/web/src/hidden.md", "# Hidden\n")
	writeDocFile(t, root, "packages/ui/hidden.md", "# Hidden\n")

	graph, diags := BuildGraph(BuildOptions{
		Root:     root,
		Entry:    "AGENTS.md",
		Excludes: pattern.NewMatcher(root, nil),
	})
	if len(diags) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diags)
	}
	if !graph.Reachable["apps/web/src"] {
		t.Fatal("apps/web/src should be recorded as a reachable existing directory")
	}
	if !graph.Reachable["packages/ui"] {
		t.Fatal("packages/ui should be recorded as a reachable existing directory")
	}
	if graph.Reachable["apps/web/src/hidden.md"] {
		t.Fatal("path-code directory references should not expand nested Markdown files")
	}
	if graph.Reachable["packages/ui/hidden.md"] {
		t.Fatal("path-text directory references should not expand nested Markdown files")
	}
}

func TestBuildGraphExpandsLinkedDirectories(t *testing.T) {
	root := t.TempDir()
	writeDocFile(t, root, "AGENTS.md", "[Docs](docs)\n")
	writeDocFile(t, root, "docs/guide.md", "# Guide\n")

	graph, diags := BuildGraph(BuildOptions{
		Root:     root,
		Entry:    "AGENTS.md",
		Excludes: pattern.NewMatcher(root, nil),
	})
	if len(diags) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diags)
	}
	if !graph.Reachable["docs/guide.md"] {
		t.Fatal("markdown links to directories should still expand nested Markdown files")
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

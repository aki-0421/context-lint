package frontmatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractReturnsOnlyOpeningFrontMatterBlock(t *testing.T) {
	got, ok := Extract([]byte("---\ntitle: Doc\n---\n# Body\n---\nnot front matter\n"))
	if !ok {
		t.Fatal("Extract() ok = false, want true")
	}
	want := "---\ntitle: Doc\n---\n"
	if got != want {
		t.Fatalf("Extract() = %q, want %q", got, want)
	}
}

func TestExtractSupportsCRLF(t *testing.T) {
	got, ok := Extract([]byte("---\r\ntitle: Doc\r\n...\r\n# Body\r\n"))
	if !ok {
		t.Fatal("Extract() ok = false, want true")
	}
	want := "---\r\ntitle: Doc\r\n...\r\n"
	if got != want {
		t.Fatalf("Extract() = %q, want %q", got, want)
	}
}

func TestExtractIgnoresNonLeadingFrontMatterLikeBlocks(t *testing.T) {
	_, ok := Extract([]byte("# Body\n\n```yaml\n---\ntitle: Example\n---\n```\n"))
	if ok {
		t.Fatal("Extract() ok = true, want false")
	}
}

func TestScanDirectoryReturnsItemsAndMissingFrontMatterDiagnostics(t *testing.T) {
	root := t.TempDir()
	writeFrontMatterTestFile(t, root, "docs/with.md", "---\ntitle: With\n---\n# Body\n")
	writeFrontMatterTestFile(t, root, "docs/nested/also.mdx", "---\ntitle: Also\n---\n# Body\n")
	writeFrontMatterTestFile(t, root, "docs/plain.markdown", "# Plain\n")
	writeFrontMatterTestFile(t, root, "docs/ignore.txt", "---\ntitle: Ignore\n---\n")

	result := ScanDirectory(root, "docs", ScanOptions{})
	if len(result.Items) != 2 {
		t.Fatalf("items = %#v, want 2 front matter items", result.Items)
	}
	if result.Items[0].File != "docs/nested/also.mdx" || result.Items[1].File != "docs/with.md" {
		t.Fatalf("items sorted = %#v, want nested then with", result.Items)
	}
	if result.Items[0].Content != "---\ntitle: Also\n---\n" {
		t.Fatalf("content = %q, want front matter only", result.Items[0].Content)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one missing front matter diagnostic", result.Diagnostics)
	}
	diag := result.Diagnostics[0]
	if diag.Code != "CL007" || diag.File != "docs/plain.markdown" || diag.Severity != "" {
		t.Fatalf("diagnostic = %#v, want CL007 before severity is applied", diag)
	}
	if !strings.Contains(diag.Fix, "context-lint config-guide") {
		t.Fatalf("diagnostic fix = %q, want config-guide hint", diag.Fix)
	}
}

func TestScanDirectoryRequiresDirectory(t *testing.T) {
	root := t.TempDir()
	writeFrontMatterTestFile(t, root, "docs/file.md", "---\ntitle: File\n---\n")

	result := ScanDirectory(root, "docs/file.md", ScanOptions{})
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one diagnostic", result.Diagnostics)
	}
	if result.Diagnostics[0].Code != "CL005" {
		t.Fatalf("diagnostic = %#v, want CL005", result.Diagnostics[0])
	}
}

func TestScanDirectorySkipsIndexMissingFrontMatterByDefault(t *testing.T) {
	root := t.TempDir()
	writeFrontMatterTestFile(t, root, "docs/index.md", "# Index\n")
	writeFrontMatterTestFile(t, root, "docs/plain.md", "# Plain\n")

	result := ScanDirectory(root, "docs", ScanOptions{})
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one diagnostic", result.Diagnostics)
	}
	if result.Diagnostics[0].File != "docs/plain.md" {
		t.Fatalf("diagnostic file = %q, want docs/plain.md", result.Diagnostics[0].File)
	}
}

func TestScanDirectorySkipsConfiguredMissingFrontMatterFileNames(t *testing.T) {
	root := t.TempDir()
	writeFrontMatterTestFile(t, root, "docs/README.md", "# Readme\n")
	writeFrontMatterTestFile(t, root, "docs/plain.md", "# Plain\n")

	result := ScanDirectory(root, "docs", ScanOptions{
		ExcludeFileNames: []string{"README.md"},
	})
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one diagnostic", result.Diagnostics)
	}
	if result.Diagnostics[0].File != "docs/plain.md" {
		t.Fatalf("diagnostic file = %q, want docs/plain.md", result.Diagnostics[0].File)
	}
}

func writeFrontMatterTestFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

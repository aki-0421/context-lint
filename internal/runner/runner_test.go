package runner

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aki-0421/context-lint/internal/diagnostic"
)

func TestRunReportsWarningsWithoutFailingByDefault(t *testing.T) {
	root := t.TempDir()
	writeRunnerFile(t, root, ".context-lint.yaml", `linter:
  document:
    entry: AGENTS.md
    requiredReachable:
      - docs
`)
	writeRunnerFile(t, root, "AGENTS.md", "[Index](docs/index.md)\n")
	writeRunnerFile(t, root, "docs/index.md", "See missing.md\n")
	writeRunnerFile(t, root, "docs/security.md", "# Security\n")

	var out bytes.Buffer
	code := Run(Options{Root: root, Format: "json", Stdout: &out})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; output: %s", code, out.String())
	}

	result := decodeResult(t, out.Bytes())
	if result.OK {
		t.Fatal("result.OK = true, want false")
	}
	assertDiagnostic(t, result.Diagnostics, "CL003", diagnostic.SeverityWarning, "missing.md")
	assertDiagnostic(t, result.Diagnostics, "CL004", diagnostic.SeverityWarning, "docs/security.md")
}

func TestRunStrictFailsOnDocumentDiagnostics(t *testing.T) {
	root := t.TempDir()
	writeRunnerFile(t, root, ".context-lint.yaml", `linter:
  document:
    entry: AGENTS.md
    requiredReachable:
      - docs/security.md
`)
	writeRunnerFile(t, root, "AGENTS.md", "# Entry\n")
	writeRunnerFile(t, root, "docs/security.md", "# Security\n")

	var out bytes.Buffer
	code := Run(Options{Root: root, Strict: true, Format: "json", Stdout: &out})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; output: %s", code, out.String())
	}

	result := decodeResult(t, out.Bytes())
	assertDiagnostic(t, result.Diagnostics, "CL004", diagnostic.SeverityError, "docs/security.md")
}

func TestRunExcludesRequiredTargets(t *testing.T) {
	root := t.TempDir()
	writeRunnerFile(t, root, ".context-lint.yaml", `linter:
  document:
    entry: AGENTS.md
    requiredReachable:
      - docs
    excludes:
      - docs/secret.md
`)
	writeRunnerFile(t, root, "AGENTS.md", "[Index](docs/index.md)\n")
	writeRunnerFile(t, root, "docs/index.md", "# Index\n")
	writeRunnerFile(t, root, "docs/secret.md", "# Secret\n")

	var out bytes.Buffer
	code := Run(Options{Root: root, Strict: true, Format: "json", Stdout: &out})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; output: %s", code, out.String())
	}
	result := decodeResult(t, out.Bytes())
	if !result.OK || len(result.Diagnostics) != 0 {
		t.Fatalf("result = %#v, want ok with no diagnostics", result)
	}
}

func TestRunReturnsConfigErrorForUnsupportedFormat(t *testing.T) {
	var out bytes.Buffer
	code := Run(Options{Root: t.TempDir(), Format: "sarif", Stdout: &out})
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if out.Len() == 0 {
		t.Fatal("expected an unsupported format diagnostic")
	}
}

func decodeResult(t *testing.T, data []byte) diagnostic.Result {
	t.Helper()
	var result diagnostic.Result
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal: %v\n%s", err, string(data))
	}
	return result
}

func assertDiagnostic(t *testing.T, diags []diagnostic.Diagnostic, code string, severity diagnostic.Severity, ref string) {
	t.Helper()
	for _, diag := range diags {
		if diag.Code == code && diag.Severity == severity && diag.Reference == ref {
			return
		}
	}
	t.Fatalf("missing diagnostic code=%s severity=%s ref=%s in %#v", code, severity, ref, diags)
}

func writeRunnerFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aki-0421/context-lint/internal/config"
	"github.com/aki-0421/context-lint/internal/diagnostic"
	"github.com/aki-0421/context-lint/internal/document"
	"github.com/aki-0421/context-lint/internal/frontmatter"
	"github.com/aki-0421/context-lint/internal/pattern"
)

type Options struct {
	ConfigPath string
	Root       string
	Strict     bool
	Format     string
	NoColor    bool
	Stdout     io.Writer
	Stderr     io.Writer
}

type ListResult struct {
	OK          bool                    `json:"ok"`
	Strict      bool                    `json:"strict"`
	Root        string                  `json:"root"`
	Path        string                  `json:"path"`
	FrontMatter []frontmatter.Item      `json:"frontMatter"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

func Run(opts Options) int {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Format == "" {
		opts.Format = "human"
	}
	if opts.Format != "human" && opts.Format != "json" {
		result := resultWithDiagnostic(opts, "", "", diagnostic.Diagnostic{
			Code:    "CL005",
			Message: fmt.Sprintf("Output format %q is not supported.", opts.Format),
			Fix:     "Use --format human or --format json.",
		})
		writeResult(opts, result)
		return 2
	}

	root, err := resolveRoot(opts.Root)
	if err != nil {
		result := resultWithDiagnostic(opts, "", "", diagnostic.Diagnostic{
			Code:    "CL005",
			Message: fmt.Sprintf("Project root is invalid: %v.", err),
			Fix:     "Pass a valid directory with --root or run context-lint from the project root.",
		})
		writeResult(opts, result)
		return 2
	}

	configPath, err := config.Discover(root, opts.ConfigPath)
	if err != nil {
		result := resultWithDiagnostic(opts, root, "", diagnostic.Diagnostic{
			Code:    "CL005",
			Message: err.Error() + ".",
			Fix:     "Create .context-lint.yaml, .context-lint.yml, .context-lint.json, or .context-lint.jsonc at the project root.",
		})
		writeResult(opts, result)
		return 2
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		result := resultWithDiagnostic(opts, root, relToRoot(root, configPath), diagnostic.Diagnostic{
			Code:    "CL005",
			File:    relToRoot(root, configPath),
			Message: fmt.Sprintf("Configuration file %s is invalid: %v.", relToRoot(root, configPath), err),
			Fix:     "Fix the configuration file format and required fields.",
		})
		writeResult(opts, result)
		return 2
	}

	docCfg := cfg.Linter.Document
	configRel := relToRoot(root, configPath)
	excludes := pattern.NewMatcher(root, docCfg.Excludes)
	entry := pattern.Normalize(docCfg.Entry)
	diags := validateEntry(root, entry, excludes)
	if len(diags) == 0 {
		required, emptyRequired, err := pattern.Expand(root, docCfg.RequiredReachable, excludes, true)
		if err != nil {
			diags = append(diags, diagnostic.Diagnostic{
				Code:    "CL005",
				File:    configRel,
				Message: fmt.Sprintf("requiredReachable could not be expanded: %v.", err),
				Fix:     "Use project-root-relative file, directory, or glob patterns.",
			})
		}
		for _, item := range emptyRequired {
			diags = append(diags, diagnostic.Diagnostic{
				Code:      "CL005",
				File:      configRel,
				Message:   fmt.Sprintf("requiredReachable item %s did not match any Markdown files.", item),
				Fix:       "Correct the pattern, create the target document, or remove the item.",
				Reference: item,
			})
		}
		if len(diags) == 0 {
			graph, graphDiags := document.BuildGraph(document.BuildOptions{
				Root:     root,
				Entry:    entry,
				Excludes: excludes,
			})
			diags = append(diags, graphDiags...)
			for _, target := range required {
				if !graph.Reachable[target] {
					diags = append(diags, diagnostic.Diagnostic{
						Code:      "CL004",
						File:      entry,
						Message:   fmt.Sprintf("requiredReachable target %s is not reachable from %s.", target, entry),
						Fix:       fmt.Sprintf("Add a path from %s to %s, directly or through an index document.", entry, target),
						Reference: target,
					})
				}
			}
		}
	}

	diagnostic.ApplySeverity(diags, opts.Strict)
	diagnostic.Sort(diags)
	result := diagnostic.Result{
		OK:          len(diags) == 0,
		Strict:      opts.Strict,
		Root:        root,
		Config:      configRel,
		Entry:       entry,
		Diagnostics: diags,
	}
	writeResult(opts, result)

	if hasConfigError(diags) {
		return 2
	}
	if opts.Strict && len(diags) > 0 {
		return 1
	}
	return 0
}

func List(opts Options, target string) int {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Format == "" {
		opts.Format = "human"
	}
	if opts.Format != "human" && opts.Format != "json" {
		result := listResultWithDiagnostic(opts, "", "", diagnostic.Diagnostic{
			Code:    "CL005",
			Message: fmt.Sprintf("Output format %q is not supported.", opts.Format),
			Fix:     "Use --format human or --format json.",
		})
		writeListResult(opts, result)
		return 2
	}

	root, err := resolveRoot(opts.Root)
	if err != nil {
		result := listResultWithDiagnostic(opts, "", "", diagnostic.Diagnostic{
			Code:    "CL005",
			Message: fmt.Sprintf("Project root is invalid: %v.", err),
			Fix:     "Pass a valid directory with --root or run context-lint from the project root.",
		})
		writeListResult(opts, result)
		return 2
	}

	excludeFileNames, configDiags := listFrontMatterExcludeFileNames(opts, root)
	if len(configDiags) > 0 {
		diagnostic.ApplySeverity(configDiags, opts.Strict)
		diagnostic.Sort(configDiags)
		result := ListResult{
			OK:          false,
			Strict:      opts.Strict,
			Root:        root,
			FrontMatter: []frontmatter.Item{},
			Diagnostics: configDiags,
		}
		writeListResult(opts, result)
		return 2
	}

	scanned := frontmatter.ScanDirectory(root, target, frontmatter.ScanOptions{
		ExcludeFileNames: excludeFileNames,
	})
	diagnostic.ApplySeverity(scanned.Diagnostics, opts.Strict)
	diagnostic.Sort(scanned.Diagnostics)
	items := scanned.Items
	if items == nil {
		items = []frontmatter.Item{}
	}
	diags := scanned.Diagnostics
	if diags == nil {
		diags = []diagnostic.Diagnostic{}
	}
	result := ListResult{
		OK:          len(diags) == 0,
		Strict:      opts.Strict,
		Root:        root,
		Path:        scanned.Path,
		FrontMatter: items,
		Diagnostics: diags,
	}
	writeListResult(opts, result)

	if hasConfigError(diags) {
		return 2
	}
	if opts.Strict && len(diags) > 0 {
		return 1
	}
	return 0
}

func resolveRoot(root string) (string, error) {
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", root)
	}
	return abs, nil
}

func validateEntry(root string, entry string, excludes pattern.Matcher) []diagnostic.Diagnostic {
	var diags []diagnostic.Diagnostic
	if entry == ".." || filepath.IsAbs(entry) || filepath.ToSlash(entry) == ".." || len(entry) >= 3 && entry[:3] == "../" {
		return append(diags, diagnostic.Diagnostic{
			Code:      "CL005",
			File:      entry,
			Message:   fmt.Sprintf("Entry file %s escapes the project root.", entry),
			Fix:       "Set linter.document.entry to a project-root-relative Markdown file.",
			Reference: entry,
		})
	}
	if excludes.Match(entry) {
		return append(diags, diagnostic.Diagnostic{
			Code:      "CL005",
			File:      entry,
			Message:   fmt.Sprintf("Entry file %s is excluded by linter.document.excludes.", entry),
			Fix:       "Remove the entry file from excludes or choose a different entry file.",
			Reference: entry,
		})
	}
	if !pattern.IsMarkdown(entry) {
		diags = append(diags, diagnostic.Diagnostic{
			Code:      "CL002",
			File:      entry,
			Message:   fmt.Sprintf("Entry file %s is not a Markdown file.", entry),
			Fix:       "Set linter.document.entry to a .md, .mdx, or .markdown file.",
			Reference: entry,
		})
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(entry)))
	if err != nil {
		diags = append(diags, diagnostic.Diagnostic{
			Code:      "CL001",
			File:      entry,
			Message:   fmt.Sprintf("Entry file %s does not exist.", entry),
			Fix:       "Create the entry file or update linter.document.entry.",
			Reference: entry,
		})
		return diags
	}
	if info.IsDir() {
		diags = append(diags, diagnostic.Diagnostic{
			Code:      "CL002",
			File:      entry,
			Message:   fmt.Sprintf("Entry file %s is a directory, but it must be a single Markdown file.", entry),
			Fix:       "Set linter.document.entry to a Markdown file.",
			Reference: entry,
		})
	}
	return diags
}

func resultWithDiagnostic(opts Options, root string, configPath string, diag diagnostic.Diagnostic) diagnostic.Result {
	diags := []diagnostic.Diagnostic{diag}
	diagnostic.ApplySeverity(diags, opts.Strict)
	return diagnostic.Result{
		OK:          false,
		Strict:      opts.Strict,
		Root:        root,
		Config:      configPath,
		Diagnostics: diags,
	}
}

func listResultWithDiagnostic(opts Options, root string, path string, diag diagnostic.Diagnostic) ListResult {
	diags := []diagnostic.Diagnostic{diag}
	diagnostic.ApplySeverity(diags, opts.Strict)
	return ListResult{
		OK:          false,
		Strict:      opts.Strict,
		Root:        root,
		Path:        path,
		FrontMatter: []frontmatter.Item{},
		Diagnostics: diags,
	}
}

func listFrontMatterExcludeFileNames(opts Options, root string) ([]string, []diagnostic.Diagnostic) {
	configPath, err := config.Discover(root, opts.ConfigPath)
	if err != nil {
		if opts.ConfigPath == "" {
			return nil, nil
		}
		return nil, []diagnostic.Diagnostic{{
			Code:    "CL005",
			Message: err.Error() + ".",
			Fix:     "Pass an existing configuration file with --config or remove the option.",
		}}
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		configRel := relToRoot(root, configPath)
		return nil, []diagnostic.Diagnostic{{
			Code:    "CL005",
			File:    configRel,
			Message: fmt.Sprintf("Configuration file %s is invalid: %v.", configRel, err),
			Fix:     "Fix the configuration file format and required fields.",
		}}
	}

	return cfg.Linter.Document.FrontMatter.ExcludeFileNames, nil
}

func writeResult(opts Options, result diagnostic.Result) {
	switch opts.Format {
	case "human":
		_ = diagnostic.WriteHuman(opts.Stdout, result)
	case "json":
		_ = diagnostic.WriteJSON(opts.Stdout, result)
	default:
		_ = diagnostic.WriteHuman(opts.Stdout, result)
	}
}

func writeListResult(opts Options, result ListResult) {
	switch opts.Format {
	case "human":
		_ = writeListHuman(opts.Stdout, result)
	case "json":
		enc := json.NewEncoder(opts.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	default:
		_ = writeListHuman(opts.Stdout, result)
	}
}

func writeListHuman(w io.Writer, result ListResult) error {
	if len(result.FrontMatter) == 0 && len(result.Diagnostics) == 0 {
		_, err := fmt.Fprintln(w, "context-lint list: no Markdown files found")
		return err
	}

	for i, item := range result.FrontMatter {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w, item.File); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, item.Content); err != nil {
			return err
		}
		if !strings.HasSuffix(item.Content, "\n") {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	}

	if len(result.Diagnostics) == 0 {
		return nil
	}
	if len(result.FrontMatter) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return diagnostic.WriteHuman(w, diagnostic.Result{
		OK:          result.OK,
		Strict:      result.Strict,
		Root:        result.Root,
		Diagnostics: result.Diagnostics,
	})
}

func hasConfigError(diags []diagnostic.Diagnostic) bool {
	for _, diag := range diags {
		if diag.Code == "CL005" {
			return true
		}
	}
	return false
}

func relToRoot(root string, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(rel)
}

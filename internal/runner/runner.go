package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
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

type ManagedCheckResult struct {
	Managed     bool                    `json:"managed"`
	Root        string                  `json:"root"`
	Config      string                  `json:"config,omitempty"`
	Path        string                  `json:"path"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

type ManagedListResult struct {
	Root        string                  `json:"root"`
	Config      string                  `json:"config,omitempty"`
	Path        string                  `json:"path"`
	Files       []string                `json:"files"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

type ManagedTreeResult struct {
	Root        string                  `json:"root"`
	Config      string                  `json:"config,omitempty"`
	Path        string                  `json:"path"`
	Tree        *ManagedTreeNode        `json:"tree,omitempty"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

type ManagedTreeNode struct {
	Name     string            `json:"name"`
	Path     string            `json:"path"`
	Type     string            `json:"type"`
	Children []ManagedTreeNode `json:"children,omitempty"`
}

type managedTargets struct {
	root   string
	config string
	items  []string
	set    map[string]bool
}

type managedTreeBuildNode struct {
	item     ManagedTreeNode
	children map[string]*managedTreeBuildNode
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
			diags = append(diags, checkRequiredReachableFileSizes(root, entry, required, docCfg.RequiredReachableMaxFileSize.EffectiveBytes(config.DefaultRequiredReachableMaxFileSizeBytes))...)
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

func ManagedCheck(opts Options, target string) int {
	opts = normalizeCommandOptions(opts)
	formatDiag := validateOutputFormat(opts.Format)
	if formatDiag != nil {
		diags := managedDiagnostics([]diagnostic.Diagnostic{*formatDiag}, opts.Strict)
		writeManagedCheckResult(opts, ManagedCheckResult{Diagnostics: diags})
		return 2
	}

	root, diags := resolveManagedRoot(opts)
	if len(diags) > 0 {
		writeManagedCheckResult(opts, ManagedCheckResult{Diagnostics: diags})
		return 2
	}

	rel, diags := normalizeManagedInputPath(root, target, "Managed check path", "Run context-lint managed check <file>.")
	if len(diags) > 0 {
		writeManagedCheckResult(opts, ManagedCheckResult{
			Root:        root,
			Path:        displayManagedPath(rel),
			Diagnostics: diags,
		})
		return 2
	}

	targets, diags := loadManagedTargets(opts, root)
	if len(diags) > 0 {
		writeManagedCheckResult(opts, ManagedCheckResult{
			Root:        root,
			Config:      targets.config,
			Path:        displayManagedPath(rel),
			Diagnostics: diags,
		})
		return 2
	}

	managed := rel != "" && targets.set[rel]
	writeManagedCheckResult(opts, ManagedCheckResult{
		Managed:     managed,
		Root:        root,
		Config:      targets.config,
		Path:        displayManagedPath(rel),
		Diagnostics: []diagnostic.Diagnostic{},
	})
	if managed {
		return 0
	}
	return 1
}

func ManagedList(opts Options, target string) int {
	opts = normalizeCommandOptions(opts)
	formatDiag := validateOutputFormat(opts.Format)
	if formatDiag != nil {
		diags := managedDiagnostics([]diagnostic.Diagnostic{*formatDiag}, opts.Strict)
		writeManagedListResult(opts, ManagedListResult{Diagnostics: diags})
		return 2
	}

	root, diags := resolveManagedRoot(opts)
	if len(diags) > 0 {
		writeManagedListResult(opts, ManagedListResult{Diagnostics: diags})
		return 2
	}

	rel, diags := normalizeManagedInputPath(root, target, "Managed list path", "Run context-lint managed list <directory>.")
	if len(diags) > 0 {
		writeManagedListResult(opts, ManagedListResult{
			Root:        root,
			Path:        displayManagedPath(rel),
			Files:       []string{},
			Diagnostics: diags,
		})
		return 2
	}

	info, diags := statManagedInputPath(root, rel, "Managed list path", "Pass an existing directory path.")
	if len(diags) > 0 {
		writeManagedListResult(opts, ManagedListResult{
			Root:        root,
			Path:        displayManagedPath(rel),
			Files:       []string{},
			Diagnostics: diags,
		})
		return 2
	}
	if !info.IsDir() {
		diags := managedDiagnostics([]diagnostic.Diagnostic{{
			Code:      "CL005",
			File:      displayManagedPath(rel),
			Message:   fmt.Sprintf("Managed list path %s is not a directory.", displayManagedPath(rel)),
			Fix:       "Pass a directory path.",
			Reference: target,
		}}, opts.Strict)
		writeManagedListResult(opts, ManagedListResult{
			Root:        root,
			Path:        displayManagedPath(rel),
			Files:       []string{},
			Diagnostics: diags,
		})
		return 2
	}

	targets, diags := loadManagedTargets(opts, root)
	if len(diags) > 0 {
		writeManagedListResult(opts, ManagedListResult{
			Root:        root,
			Config:      targets.config,
			Path:        displayManagedPath(rel),
			Files:       []string{},
			Diagnostics: diags,
		})
		return 2
	}

	files := directManagedFiles(targets.items, rel)
	writeManagedListResult(opts, ManagedListResult{
		Root:        root,
		Config:      targets.config,
		Path:        displayManagedPath(rel),
		Files:       files,
		Diagnostics: []diagnostic.Diagnostic{},
	})
	return 0
}

func ManagedTree(opts Options, target string) int {
	opts = normalizeCommandOptions(opts)
	formatDiag := validateOutputFormat(opts.Format)
	if formatDiag != nil {
		diags := managedDiagnostics([]diagnostic.Diagnostic{*formatDiag}, opts.Strict)
		writeManagedTreeResult(opts, ManagedTreeResult{Diagnostics: diags})
		return 2
	}

	root, diags := resolveManagedRoot(opts)
	if len(diags) > 0 {
		writeManagedTreeResult(opts, ManagedTreeResult{Diagnostics: diags})
		return 2
	}

	rel, diags := normalizeManagedInputPath(root, target, "Managed tree path", "Run context-lint managed tree <path>.")
	if len(diags) > 0 {
		writeManagedTreeResult(opts, ManagedTreeResult{
			Root:        root,
			Path:        displayManagedPath(rel),
			Diagnostics: diags,
		})
		return 2
	}

	info, diags := statManagedInputPath(root, rel, "Managed tree path", "Pass an existing file or directory path.")
	if len(diags) > 0 {
		writeManagedTreeResult(opts, ManagedTreeResult{
			Root:        root,
			Path:        displayManagedPath(rel),
			Diagnostics: diags,
		})
		return 2
	}

	targets, diags := loadManagedTargets(opts, root)
	if len(diags) > 0 {
		writeManagedTreeResult(opts, ManagedTreeResult{
			Root:        root,
			Config:      targets.config,
			Path:        displayManagedPath(rel),
			Diagnostics: diags,
		})
		return 2
	}

	tree := buildManagedTree(rel, info.IsDir(), targets.items)
	writeManagedTreeResult(opts, ManagedTreeResult{
		Root:        root,
		Config:      targets.config,
		Path:        displayManagedPath(rel),
		Tree:        tree,
		Diagnostics: []diagnostic.Diagnostic{},
	})
	return 0
}

func normalizeCommandOptions(opts Options) Options {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Format == "" {
		opts.Format = "human"
	}
	return opts
}

func validateOutputFormat(format string) *diagnostic.Diagnostic {
	if format == "human" || format == "json" {
		return nil
	}
	return &diagnostic.Diagnostic{
		Code:    "CL005",
		Message: fmt.Sprintf("Output format %q is not supported.", format),
		Fix:     "Use --format human or --format json.",
	}
}

func resolveManagedRoot(opts Options) (string, []diagnostic.Diagnostic) {
	root, err := resolveRoot(opts.Root)
	if err == nil {
		return root, nil
	}
	return "", managedDiagnostics([]diagnostic.Diagnostic{{
		Code:    "CL005",
		Message: fmt.Sprintf("Project root is invalid: %v.", err),
		Fix:     "Pass a valid directory with --root or run context-lint from the project root.",
	}}, opts.Strict)
}

func normalizeManagedInputPath(root string, target string, label string, fix string) (string, []diagnostic.Diagnostic) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", managedDiagnostics([]diagnostic.Diagnostic{{
			Code:    "CL005",
			Message: label + " is required.",
			Fix:     fix,
		}}, false)
	}

	input := filepath.FromSlash(target)
	abs := input
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, input)
	}
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", managedDiagnostics([]diagnostic.Diagnostic{{
			Code:      "CL005",
			Message:   fmt.Sprintf("%s %s is invalid: %v.", label, target, err),
			Fix:       fix,
			Reference: target,
		}}, false)
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", managedDiagnostics([]diagnostic.Diagnostic{{
			Code:      "CL005",
			Message:   fmt.Sprintf("%s %s is invalid: %v.", label, target, err),
			Fix:       fix,
			Reference: target,
		}}, false)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "", nil
	}
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", managedDiagnostics([]diagnostic.Diagnostic{{
			Code:      "CL005",
			Message:   fmt.Sprintf("%s %s escapes the project root.", label, target),
			Fix:       "Pass a project-root-relative path or a path under --root.",
			Reference: target,
		}}, false)
	}

	return pattern.Normalize(rel), nil
}

func statManagedInputPath(root string, rel string, label string, fix string) (os.FileInfo, []diagnostic.Diagnostic) {
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	if err == nil {
		return info, nil
	}
	return nil, managedDiagnostics([]diagnostic.Diagnostic{{
		Code:      "CL005",
		File:      displayManagedPath(rel),
		Message:   fmt.Sprintf("%s %s does not exist.", label, displayManagedPath(rel)),
		Fix:       fix,
		Reference: displayManagedPath(rel),
	}}, false)
}

func loadManagedTargets(opts Options, root string) (managedTargets, []diagnostic.Diagnostic) {
	configPath, err := config.Discover(root, opts.ConfigPath)
	if err != nil {
		return managedTargets{root: root, set: map[string]bool{}}, managedDiagnostics([]diagnostic.Diagnostic{{
			Code:    "CL005",
			Message: err.Error() + ".",
			Fix:     "Create .context-lint.yaml, .context-lint.yml, .context-lint.json, or .context-lint.jsonc at the project root.",
		}}, opts.Strict)
	}

	cfg, err := config.Load(configPath)
	configRel := relToRoot(root, configPath)
	if err != nil {
		return managedTargets{root: root, config: configRel, set: map[string]bool{}}, managedDiagnostics([]diagnostic.Diagnostic{{
			Code:    "CL005",
			File:    configRel,
			Message: fmt.Sprintf("Configuration file %s is invalid: %v.", configRel, err),
			Fix:     "Fix the configuration file format and required fields.",
		}}, opts.Strict)
	}

	docCfg := cfg.Linter.Document
	excludes := pattern.NewMatcher(root, docCfg.Excludes)
	required, emptyRequired, err := pattern.Expand(root, docCfg.RequiredReachable, excludes, true)
	if err != nil {
		return managedTargets{root: root, config: configRel, set: map[string]bool{}}, managedDiagnostics([]diagnostic.Diagnostic{{
			Code:    "CL005",
			File:    configRel,
			Message: fmt.Sprintf("requiredReachable could not be expanded: %v.", err),
			Fix:     "Use project-root-relative file, directory, or glob patterns.",
		}}, opts.Strict)
	}

	var diags []diagnostic.Diagnostic
	for _, item := range emptyRequired {
		diags = append(diags, diagnostic.Diagnostic{
			Code:      "CL005",
			File:      configRel,
			Message:   fmt.Sprintf("requiredReachable item %s did not match any Markdown files.", item),
			Fix:       "Correct the pattern, create the target document, or remove the item.",
			Reference: item,
		})
	}
	if len(diags) > 0 {
		return managedTargets{root: root, config: configRel, set: map[string]bool{}}, managedDiagnostics(diags, opts.Strict)
	}

	set := make(map[string]bool, len(required))
	for _, item := range required {
		set[item] = true
	}
	return managedTargets{
		root:   root,
		config: configRel,
		items:  required,
		set:    set,
	}, nil
}

func directManagedFiles(targets []string, dir string) []string {
	var files []string
	for _, target := range targets {
		if parentManagedPath(target) == dir {
			files = append(files, target)
		}
	}
	return files
}

func parentManagedPath(file string) string {
	parent := path.Dir(file)
	if parent == "." {
		return ""
	}
	return parent
}

func buildManagedTree(targetRel string, targetIsDir bool, targets []string) *ManagedTreeNode {
	if !targetIsDir {
		for _, target := range targets {
			if target == targetRel {
				return &ManagedTreeNode{
					Name: displayManagedPath(target),
					Path: target,
					Type: "file",
				}
			}
		}
		return nil
	}

	root := &managedTreeBuildNode{
		item: ManagedTreeNode{
			Name: treeRootName(targetRel),
			Path: displayManagedPath(targetRel),
			Type: "directory",
		},
		children: map[string]*managedTreeBuildNode{},
	}

	for _, target := range targets {
		if !managedTreeContains(targetRel, target) {
			continue
		}
		suffix := target
		if targetRel != "" {
			suffix = strings.TrimPrefix(target, targetRel+"/")
		}
		addManagedTreePath(root, targetRel, strings.Split(suffix, "/"))
	}

	item := flattenManagedTree(root)
	return &item
}

func managedTreeContains(root string, target string) bool {
	if root == "" {
		return true
	}
	return strings.HasPrefix(target, root+"/")
}

func addManagedTreePath(root *managedTreeBuildNode, rootRel string, parts []string) {
	current := root
	currentPath := rootRel
	for i, part := range parts {
		if part == "" {
			continue
		}
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath += "/" + part
		}
		nodeType := "directory"
		if i == len(parts)-1 {
			nodeType = "file"
		}
		current = current.child(part, currentPath, nodeType)
	}
}

func (n *managedTreeBuildNode) child(name string, rel string, nodeType string) *managedTreeBuildNode {
	if n.children == nil {
		n.children = map[string]*managedTreeBuildNode{}
	}
	if existing, ok := n.children[name]; ok {
		return existing
	}
	child := &managedTreeBuildNode{
		item: ManagedTreeNode{
			Name: name,
			Path: rel,
			Type: nodeType,
		},
		children: map[string]*managedTreeBuildNode{},
	}
	n.children[name] = child
	return child
}

func flattenManagedTree(node *managedTreeBuildNode) ManagedTreeNode {
	item := node.item
	keys := make([]string, 0, len(node.children))
	for key := range node.children {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		item.Children = append(item.Children, flattenManagedTree(node.children[key]))
	}
	return item
}

func treeRootName(rel string) string {
	return displayManagedPath(rel)
}

func displayManagedPath(rel string) string {
	if rel == "" {
		return "."
	}
	return rel
}

func managedDiagnostics(diags []diagnostic.Diagnostic, strict bool) []diagnostic.Diagnostic {
	if diags == nil {
		return []diagnostic.Diagnostic{}
	}
	diagnostic.ApplySeverity(diags, strict)
	diagnostic.Sort(diags)
	return diags
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

func checkRequiredReachableFileSizes(root string, entry string, required []string, maxBytes int64) []diagnostic.Diagnostic {
	var diags []diagnostic.Diagnostic
	for _, target := range required {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(target)))
		if err != nil || info.IsDir() || info.Size() < maxBytes {
			continue
		}
		diags = append(diags, diagnostic.Diagnostic{
			Code:      "CL008",
			File:      target,
			Message:   fmt.Sprintf("requiredReachable target %s is %s, at or above the %s limit.", target, formatFileSize(info.Size()), formatFileSize(maxBytes)),
			Fix:       fmt.Sprintf("Split %s into smaller focused Markdown files, then link them from %s or an index document so agents and humans can read the context progressively.", target, entry),
			Reference: target,
		})
	}
	return diags
}

func formatFileSize(bytes int64) string {
	const (
		kib = int64(1024)
		mib = kib * 1024
		gib = mib * 1024
	)
	switch {
	case bytes >= gib:
		return formatScaledFileSize(bytes, gib, "GiB")
	case bytes >= mib:
		return formatScaledFileSize(bytes, mib, "MiB")
	case bytes >= kib:
		return formatScaledFileSize(bytes, kib, "KiB")
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatScaledFileSize(bytes int64, unit int64, suffix string) string {
	if bytes%unit == 0 {
		return fmt.Sprintf("%d %s", bytes/unit, suffix)
	}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(unit), suffix)
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

func writeManagedCheckResult(opts Options, result ManagedCheckResult) {
	switch opts.Format {
	case "human":
		_ = writeManagedCheckHuman(opts.Stdout, result)
	case "json":
		enc := json.NewEncoder(opts.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	default:
		_ = writeManagedCheckHuman(opts.Stdout, result)
	}
}

func writeManagedListResult(opts Options, result ManagedListResult) {
	switch opts.Format {
	case "human":
		_ = writeManagedListHuman(opts.Stdout, result)
	case "json":
		enc := json.NewEncoder(opts.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	default:
		_ = writeManagedListHuman(opts.Stdout, result)
	}
}

func writeManagedTreeResult(opts Options, result ManagedTreeResult) {
	switch opts.Format {
	case "human":
		_ = writeManagedTreeHuman(opts.Stdout, result)
	case "json":
		enc := json.NewEncoder(opts.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	default:
		_ = writeManagedTreeHuman(opts.Stdout, result)
	}
}

func writeManagedCheckHuman(w io.Writer, result ManagedCheckResult) error {
	if len(result.Diagnostics) > 0 {
		return diagnostic.WriteHuman(w, diagnostic.Result{
			OK:          false,
			Root:        result.Root,
			Config:      result.Config,
			Diagnostics: result.Diagnostics,
		})
	}
	_, err := fmt.Fprintln(w, result.Managed)
	return err
}

func writeManagedListHuman(w io.Writer, result ManagedListResult) error {
	if len(result.Diagnostics) > 0 {
		return diagnostic.WriteHuman(w, diagnostic.Result{
			OK:          false,
			Root:        result.Root,
			Config:      result.Config,
			Diagnostics: result.Diagnostics,
		})
	}
	for _, file := range result.Files {
		if _, err := fmt.Fprintln(w, file); err != nil {
			return err
		}
	}
	return nil
}

func writeManagedTreeHuman(w io.Writer, result ManagedTreeResult) error {
	if len(result.Diagnostics) > 0 {
		return diagnostic.WriteHuman(w, diagnostic.Result{
			OK:          false,
			Root:        result.Root,
			Config:      result.Config,
			Diagnostics: result.Diagnostics,
		})
	}
	if result.Tree == nil {
		return nil
	}
	return writeManagedTreeNode(w, *result.Tree)
}

func writeManagedTreeNode(w io.Writer, node ManagedTreeNode) error {
	if _, err := fmt.Fprintln(w, node.Name); err != nil {
		return err
	}
	for i, child := range node.Children {
		if err := writeManagedTreeChild(w, child, "", i == len(node.Children)-1); err != nil {
			return err
		}
	}
	return nil
}

func writeManagedTreeChild(w io.Writer, node ManagedTreeNode, prefix string, last bool) error {
	connector := "├── "
	nextPrefix := prefix + "│   "
	if last {
		connector = "└── "
		nextPrefix = prefix + "    "
	}
	if _, err := fmt.Fprintf(w, "%s%s%s\n", prefix, connector, node.Name); err != nil {
		return err
	}
	for i, child := range node.Children {
		if err := writeManagedTreeChild(w, child, nextPrefix, i == len(node.Children)-1); err != nil {
			return err
		}
	}
	return nil
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

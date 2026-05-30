package document

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/aki-0421/context-lint/internal/diagnostic"
	"github.com/aki-0421/context-lint/internal/pattern"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

const (
	KindMarkdownLink = "markdown-link"
	KindHTMLAttr     = "html-attr"
	KindPathText     = "path-text"
	KindPathCode     = "path-code"
)

type Reference struct {
	SourceRel string
	TargetRaw string
	TargetRel string
	Kind      string
	Line      int
	Column    int
}

type Graph struct {
	Reachable  map[string]bool
	References []Reference
}

type BuildOptions struct {
	Root     string
	Entry    string
	Excludes pattern.Matcher
}

type resolvedReference struct {
	external bool
	escaped  bool
	glob     bool
	targets  []string
}

var (
	htmlAttrRE = regexp.MustCompile(`(?i)\b(?:href|src)\s*=\s*["']([^"']+)["']`)
	pathLikeRE = regexp.MustCompile(`(?i)(^|[\s'"(<\[])(` +
		`(?:\.{1,2}/|/)?[A-Za-z0-9_.@~+*?\[\]{}=-]+(?:[\\/][A-Za-z0-9_.@~+*?\[\]{}=-]+)+(?:#[A-Za-z0-9_.~/%-]+)?` +
		`|[A-Za-z0-9_.@~+-]+\.(?:md|mdx|markdown|txt|json|jsonc|ya?ml|png|jpe?g|gif|svg|webp|pdf)(?:#[A-Za-z0-9_.~/%-]+)?` +
		`)($|[\s'",).>\]])`)
	schemeRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
)

func BuildGraph(opts BuildOptions) (Graph, []diagnostic.Diagnostic) {
	graph := Graph{
		Reachable:  map[string]bool{opts.Entry: true},
		References: nil,
	}
	var diagnostics []diagnostic.Diagnostic
	visited := map[string]bool{}
	queue := []string{opts.Entry}
	missingSeen := map[string]bool{}
	escapeSeen := map[string]bool{}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		current = pattern.Normalize(current)
		if current == "" || visited[current] || opts.Excludes.Match(current) {
			continue
		}
		visited[current] = true

		data, err := os.ReadFile(filepath.Join(opts.Root, filepath.FromSlash(current)))
		if err != nil {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{
				Code:    "CL003",
				File:    current,
				Message: fmt.Sprintf("%s is reachable from %s, but it could not be read: %v.", current, opts.Entry, err),
				Fix:     "Restore the file, fix permissions, or remove the reference.",
			})
			continue
		}

		refs := ExtractReferences(current, data)
		for _, ref := range refs {
			resolved := resolveReference(opts.Root, current, ref.TargetRaw, ref.Kind)
			if resolved.external {
				continue
			}
			if resolved.escaped {
				key := fmt.Sprintf("%s:%d:%s", ref.SourceRel, ref.Line, ref.TargetRaw)
				if !escapeSeen[key] {
					escapeSeen[key] = true
					diagnostics = append(diagnostics, diagnostic.Diagnostic{
						Code:      "CL006",
						File:      ref.SourceRel,
						Line:      ref.Line,
						Column:    ref.Column,
						Message:   fmt.Sprintf("%s references %s, but that path escapes the project root.", ref.SourceRel, ref.TargetRaw),
						Fix:       "Use a project-local path or remove the reference.",
						Reference: ref.TargetRaw,
						Kind:      ref.Kind,
					})
				}
				continue
			}
			if len(resolved.targets) == 0 {
				if !shouldReportMissingReference(ref.TargetRaw) {
					continue
				}
				key := fmt.Sprintf("%s:%d:%s", ref.SourceRel, ref.Line, ref.TargetRaw)
				if !missingSeen[key] {
					missingSeen[key] = true
					diagnostics = append(diagnostics, diagnostic.Diagnostic{
						Code:      "CL003",
						File:      ref.SourceRel,
						Line:      ref.Line,
						Column:    ref.Column,
						Message:   fmt.Sprintf("%s references %s, but that file does not exist.", ref.SourceRel, ref.TargetRaw),
						Fix:       fmt.Sprintf("Create %s, correct the path, or remove the reference.", ref.TargetRaw),
						Reference: ref.TargetRaw,
						Kind:      ref.Kind,
					})
				}
				continue
			}

			for _, target := range resolved.targets {
				if opts.Excludes.Match(target) {
					continue
				}
				ref.TargetRel = target
				graph.References = append(graph.References, ref)
				graph.Reachable[target] = true
				if pattern.IsMarkdown(target) && !visited[target] {
					queue = append(queue, target)
				}
			}
		}
	}

	sort.SliceStable(graph.References, func(i, j int) bool {
		if graph.References[i].SourceRel != graph.References[j].SourceRel {
			return graph.References[i].SourceRel < graph.References[j].SourceRel
		}
		if graph.References[i].Line != graph.References[j].Line {
			return graph.References[i].Line < graph.References[j].Line
		}
		return graph.References[i].TargetRel < graph.References[j].TargetRel
	})

	return graph, diagnostics
}

func ExtractReferences(sourceRel string, data []byte) []Reference {
	var refs []Reference
	seen := map[string]bool{}
	add := func(raw string, kind string, line int, column int) {
		raw = cleanRawReference(raw)
		if raw == "" {
			return
		}
		if shouldIgnorePathLikeReference(raw, kind) {
			return
		}
		keyNoKind := fmt.Sprintf("%d:%d:%s", line, column, raw)
		if kind == KindPathText || kind == KindPathCode {
			if seen[keyNoKind] {
				return
			}
		}
		key := keyNoKind + ":" + kind
		if seen[key] {
			return
		}
		seen[key] = true
		if kind == KindMarkdownLink || kind == KindHTMLAttr {
			seen[keyNoKind] = true
		}
		refs = append(refs, Reference{
			SourceRel: sourceRel,
			TargetRaw: raw,
			Kind:      kind,
			Line:      line,
			Column:    column,
		})
	}

	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(data))
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Link:
			line, col := findPosition(data, n.Destination)
			add(string(n.Destination), KindMarkdownLink, line, col)
		case *ast.Image:
			line, col := findPosition(data, n.Destination)
			add(string(n.Destination), KindMarkdownLink, line, col)
		}
		return ast.WalkContinue, nil
	})

	lines := bytes.Split(data, []byte("\n"))
	inFence := false
	fenceMarker := ""
	for i, lineBytes := range lines {
		lineNo := i + 1
		line := string(lineBytes)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if !inFence {
				inFence = true
				fenceMarker = marker
			} else if marker == fenceMarker {
				inFence = false
				fenceMarker = ""
			}
		}

		for _, match := range htmlAttrRE.FindAllStringSubmatchIndex(line, -1) {
			if len(match) < 4 {
				continue
			}
			raw := line[match[2]:match[3]]
			add(raw, KindHTMLAttr, lineNo, match[2]+1)
		}

		kind := KindPathText
		if inFence {
			kind = KindPathCode
		}
		for _, match := range pathLikeRE.FindAllStringSubmatchIndex(line, -1) {
			if len(match) < 6 {
				continue
			}
			raw := line[match[4]:match[5]]
			add(raw, kind, lineNo, match[4]+1)
		}
	}

	return refs
}

func resolveReference(root string, sourceRel string, raw string, kind string) resolvedReference {
	target := cleanRawReference(raw)
	if target == "" || strings.HasPrefix(target, "#") || isExternal(target) {
		return resolvedReference{external: true}
	}

	target = stripAnchor(target)
	if decoded, err := url.PathUnescape(target); err == nil {
		target = decoded
	}
	target = strings.ReplaceAll(target, "\\", "/")

	baseDir := path.Dir(sourceRel)
	if baseDir == "." {
		baseDir = ""
	}

	var rel string
	if strings.HasPrefix(target, "/") {
		rel = strings.TrimPrefix(target, "/")
	} else {
		rel = path.Join(baseDir, target)
	}
	rel = path.Clean(rel)
	if rel == "." {
		rel = ""
	}
	if rel == "" || rel == ".." || strings.HasPrefix(rel, "../") {
		return resolvedReference{escaped: true}
	}
	if isDomainLikePath(rel) {
		return resolvedReference{external: true}
	}

	if pattern.HasMeta(rel) {
		matches, err := doublestar.Glob(os.DirFS(root), rel)
		if (err != nil || len(matches) == 0) && isPathTextKind(kind) && baseDir != "" && !isExplicitRelativeOrRoot(target) {
			rootRel := path.Clean(target)
			if rootRel != "." && rootRel != ".." && !strings.HasPrefix(rootRel, "../") {
				if rootMatches, rootErr := doublestar.Glob(os.DirFS(root), rootRel); rootErr == nil && len(rootMatches) > 0 {
					rel = rootRel
					matches = rootMatches
					err = nil
				}
			}
		}
		if err != nil || len(matches) == 0 {
			return resolvedReference{glob: true}
		}
		targets := make([]string, 0, len(matches))
		for _, match := range matches {
			match = pattern.Normalize(match)
			info, err := os.Stat(filepath.Join(root, filepath.FromSlash(match)))
			if err != nil {
				continue
			}
			if info.IsDir() {
				if !shouldExpandDirectory(kind) {
					targets = append(targets, match)
					continue
				}
				files, err := markdownFilesUnder(root, match)
				if err == nil {
					targets = append(targets, files...)
				}
				continue
			}
			targets = append(targets, match)
		}
		sort.Strings(targets)
		return resolvedReference{glob: true, targets: targets}
	}

	abs := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	if err != nil {
		if isPathTextKind(kind) && baseDir != "" && !isExplicitRelativeOrRoot(target) {
			rootRel := path.Clean(target)
			if rootRel != "." && rootRel != ".." && !strings.HasPrefix(rootRel, "../") {
				if rootInfo, rootErr := os.Stat(filepath.Join(root, filepath.FromSlash(rootRel))); rootErr == nil {
					rel = rootRel
					info = rootInfo
					err = nil
				}
			}
		}
	}
	if err != nil {
		if shouldIgnoreMissingExtensionless(target) {
			return resolvedReference{external: true}
		}
		return resolvedReference{}
	}
	if info.IsDir() {
		if !shouldExpandDirectory(kind) {
			return resolvedReference{targets: []string{rel}}
		}
		files, err := markdownFilesUnder(root, rel)
		if err != nil {
			return resolvedReference{}
		}
		if len(files) == 0 {
			return resolvedReference{targets: []string{rel}}
		}
		return resolvedReference{targets: files}
	}
	return resolvedReference{targets: []string{rel}}
}

func markdownFilesUnder(root string, dirRel string) ([]string, error) {
	var files []string
	dirAbs := filepath.Join(root, filepath.FromSlash(dirRel))
	err := filepath.WalkDir(dirAbs, func(current string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		rel = pattern.Normalize(rel)
		if pattern.IsMarkdown(rel) {
			files = append(files, rel)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func cleanRawReference(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "`\"'")
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	raw = strings.TrimPrefix(raw, "<")
	raw = strings.TrimSuffix(raw, ">")
	raw = strings.TrimRight(raw, ".,;:!?")
	raw = strings.TrimRight(raw, ")")
	return raw
}

func shouldIgnorePathLikeReference(raw string, kind string) bool {
	if !isPathTextKind(kind) {
		return false
	}
	switch raw {
	case "./", "../", "/", "./...", "../...":
		return true
	default:
		return strings.HasSuffix(raw, "/...")
	}
}

func stripAnchor(raw string) string {
	if i := strings.Index(raw, "#"); i >= 0 {
		return raw[:i]
	}
	return raw
}

func isExternal(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return true
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "//") {
		return true
	}
	if schemeRE.MatchString(raw) {
		return true
	}
	return false
}

func isDomainLikePath(rel string) bool {
	first := rel
	if i := strings.Index(first, "/"); i >= 0 {
		first = first[:i]
	}
	if strings.HasPrefix(first, ".") || pattern.HasMeta(first) {
		return false
	}
	if hasKnownFileExtension(first) {
		return false
	}
	return strings.Contains(first, ".")
}

func hasKnownFileExtension(value string) bool {
	switch strings.ToLower(path.Ext(value)) {
	case ".md", ".mdx", ".markdown", ".txt", ".json", ".jsonc", ".yaml", ".yml", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".pdf":
		return true
	default:
		return false
	}
}

func isPathTextKind(kind string) bool {
	return kind == KindPathText || kind == KindPathCode
}

func shouldExpandDirectory(kind string) bool {
	return kind == KindMarkdownLink || kind == KindHTMLAttr
}

func shouldReportMissingReference(raw string) bool {
	target := cleanRawReference(raw)
	if target == "" {
		return false
	}
	target = stripAnchor(target)
	if decoded, err := url.PathUnescape(target); err == nil {
		target = decoded
	}
	target = strings.ReplaceAll(target, "\\", "/")
	if pattern.IsMarkdown(target) {
		return true
	}
	if !pattern.HasMeta(target) {
		return false
	}
	switch strings.ToLower(path.Ext(target)) {
	case ".md", ".mdx", ".markdown":
		return true
	default:
		return false
	}
}

func isExplicitRelativeOrRoot(value string) bool {
	return strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") || strings.HasPrefix(value, "/")
}

func shouldIgnoreMissingExtensionless(value string) bool {
	if pattern.HasMeta(value) {
		return false
	}
	if hasKnownFileExtension(value) {
		return false
	}
	return !isExplicitRelativeOrRoot(value)
}

func findPosition(data []byte, needle []byte) (int, int) {
	if len(needle) == 0 {
		return 0, 0
	}
	idx := bytes.Index(data, needle)
	if idx < 0 {
		return 0, 0
	}
	line := 1
	col := 1
	for _, b := range data[:idx] {
		if b == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

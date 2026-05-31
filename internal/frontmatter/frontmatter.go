package frontmatter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aki-0421/context-lint/internal/diagnostic"
	"github.com/aki-0421/context-lint/internal/pattern"
)

type Item struct {
	File    string `json:"file"`
	Content string `json:"content"`
}

type ScanResult struct {
	Path        string
	Items       []Item
	Diagnostics []diagnostic.Diagnostic
}

type ScanOptions struct {
	ExcludeFileNames []string
}

func ScanDirectory(root string, target string, opts ScanOptions) ScanResult {
	var result ScanResult
	excludeFileNames := filenameSet(opts.ExcludeFileNames)
	target = strings.TrimSpace(target)
	if target == "" {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Code:    "CL005",
			Message: "List path is required.",
			Fix:     "Run context-lint list <directory>.",
		})
		return result
	}

	targetAbs := target
	if !filepath.IsAbs(targetAbs) {
		targetAbs = filepath.Join(root, filepath.FromSlash(target))
	}
	targetAbs, err := filepath.Abs(targetAbs)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Code:    "CL005",
			Message: fmt.Sprintf("List path %s is invalid: %v.", target, err),
			Fix:     "Pass a valid directory path.",
		})
		return result
	}
	result.Path = displayPath(root, targetAbs)

	info, err := os.Stat(targetAbs)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Code:      "CL005",
			File:      result.Path,
			Message:   fmt.Sprintf("List path %s does not exist.", result.Path),
			Fix:       "Pass an existing directory path.",
			Reference: target,
		})
		return result
	}
	if !info.IsDir() {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Code:      "CL005",
			File:      result.Path,
			Message:   fmt.Sprintf("List path %s is not a directory.", result.Path),
			Fix:       "Pass a directory containing Markdown files.",
			Reference: target,
		})
		return result
	}

	var files []string
	err = filepath.WalkDir(targetAbs, func(current string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if pattern.IsMarkdown(current) {
			files = append(files, current)
		}
		return nil
	})
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Code:      "CL005",
			File:      result.Path,
			Message:   fmt.Sprintf("List path %s could not be scanned: %v.", result.Path, err),
			Fix:       "Fix directory permissions or pass a readable directory.",
			Reference: target,
		})
		return result
	}
	sort.Strings(files)

	for _, file := range files {
		rel := displayPath(root, file)
		data, err := os.ReadFile(file)
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Code:      "CL005",
				File:      rel,
				Message:   fmt.Sprintf("%s could not be read: %v.", rel, err),
				Fix:       "Fix file permissions or remove the unreadable file.",
				Reference: rel,
			})
			continue
		}

		content, ok := Extract(data)
		if !ok {
			if excludeFileNames[filepath.Base(file)] {
				continue
			}
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Code:      "CL007",
				File:      rel,
				Message:   fmt.Sprintf("%s does not have front matter.", rel),
				Fix:       "Add YAML front matter at the top of the file, or run context-lint config-guide to see how to exclude file names from this check.",
				Reference: rel,
				Kind:      "front-matter",
			})
			continue
		}

		result.Items = append(result.Items, Item{
			File:    rel,
			Content: content,
		})
	}

	return result
}

func filenameSet(names []string) map[string]bool {
	set := map[string]bool{
		"index.md": true,
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		name = strings.ReplaceAll(name, "\\", "/")
		name = filepath.Base(filepath.ToSlash(name))
		if name != "." && name != "" {
			set[name] = true
		}
	}
	return set
}

func Extract(data []byte) (string, bool) {
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	firstLineEnd, next := lineBounds(data, 0)
	if firstLineEnd < 0 || strings.TrimSpace(string(data[:firstLineEnd])) != "---" {
		return "", false
	}

	offset := next
	for offset < len(data) {
		lineEnd, next := lineBounds(data, offset)
		if lineEnd < 0 {
			break
		}
		line := strings.TrimSpace(string(data[offset:lineEnd]))
		if line == "---" || line == "..." {
			return string(data[:next]), true
		}
		offset = next
	}

	return "", false
}

func lineBounds(data []byte, offset int) (int, int) {
	if offset >= len(data) {
		return -1, len(data)
	}
	for i := offset; i < len(data); i++ {
		switch data[i] {
		case '\n':
			lineEnd := i
			if lineEnd > offset && data[lineEnd-1] == '\r' {
				lineEnd--
			}
			return lineEnd, i + 1
		case '\r':
			if i+1 < len(data) && data[i+1] == '\n' {
				return i, i + 2
			}
			return i, i + 1
		}
	}
	return len(data), len(data)
}

func displayPath(root string, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "."
	}
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return filepath.ToSlash(abs)
	}
	return rel
}

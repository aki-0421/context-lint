package pattern

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type Matcher struct {
	root     string
	patterns []string
}

func NewMatcher(root string, patterns []string) Matcher {
	clean := make([]string, 0, len(patterns))
	for _, item := range patterns {
		item = Normalize(item)
		if item != "" {
			clean = append(clean, item)
		}
	}
	return Matcher{root: root, patterns: clean}
}

func Normalize(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "./")
	value = strings.TrimPrefix(value, "/")
	value = filepath.ToSlash(value)
	value = path.Clean(value)
	if value == "." {
		return ""
	}
	return value
}

func HasMeta(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func IsMarkdown(value string) bool {
	switch strings.ToLower(path.Ext(value)) {
	case ".md", ".mdx", ".markdown":
		return true
	default:
		return false
	}
}

func (m Matcher) Match(rel string) bool {
	rel = Normalize(rel)
	if rel == "" {
		return false
	}

	for _, item := range m.patterns {
		if item == rel {
			return true
		}
		if !HasMeta(item) && strings.HasPrefix(rel, item+"/") {
			return true
		}
		if HasMeta(item) {
			ok, err := doublestar.Match(item, rel)
			if err == nil && ok {
				return true
			}
		}
	}

	return false
}

func Expand(root string, items []string, excludes Matcher, markdownOnly bool) ([]string, []string, error) {
	targets := make(map[string]bool)
	var empty []string

	for _, item := range items {
		item = Normalize(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "../") || item == ".." {
			return nil, nil, fmt.Errorf("pattern %q escapes the project root", item)
		}

		added := 0
		abs := filepath.Join(root, filepath.FromSlash(item))
		info, err := os.Stat(abs)
		if err == nil {
			switch {
			case info.IsDir():
				count, err := addDirectory(root, item, excludes, markdownOnly, targets)
				if err != nil {
					return nil, nil, err
				}
				added += count
			default:
				if shouldInclude(item, excludes, markdownOnly) {
					targets[item] = true
					added++
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, nil, err
		} else if HasMeta(item) {
			matches, err := doublestar.Glob(os.DirFS(root), item)
			if err != nil {
				return nil, nil, err
			}
			for _, match := range matches {
				match = Normalize(match)
				info, err := os.Stat(filepath.Join(root, filepath.FromSlash(match)))
				if err != nil {
					return nil, nil, err
				}
				if info.IsDir() {
					count, err := addDirectory(root, match, excludes, markdownOnly, targets)
					if err != nil {
						return nil, nil, err
					}
					added += count
					continue
				}
				if shouldInclude(match, excludes, markdownOnly) {
					targets[match] = true
					added++
				}
			}
		}

		if added == 0 {
			empty = append(empty, item)
		}
	}

	out := make([]string, 0, len(targets))
	for target := range targets {
		out = append(out, target)
	}
	sort.Strings(out)
	sort.Strings(empty)
	return out, empty, nil
}

func addDirectory(root string, rel string, excludes Matcher, markdownOnly bool, targets map[string]bool) (int, error) {
	count := 0
	abs := filepath.Join(root, filepath.FromSlash(rel))
	err := filepath.WalkDir(abs, func(current string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		currentRel, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		currentRel = Normalize(currentRel)
		if currentRel == "" {
			return nil
		}
		if excludes.Match(currentRel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if markdownOnly && !IsMarkdown(currentRel) {
			return nil
		}
		targets[currentRel] = true
		count++
		return nil
	})
	return count, err
}

func shouldInclude(rel string, excludes Matcher, markdownOnly bool) bool {
	if excludes.Match(rel) {
		return false
	}
	if markdownOnly && !IsMarkdown(rel) {
		return false
	}
	return true
}

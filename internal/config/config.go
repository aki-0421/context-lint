package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var DefaultNames = []string{
	".context-lint.yaml",
	".context-lint.yml",
	".context-lint.json",
	".context-lint.jsonc",
}

type Config struct {
	Linter LinterConfig `json:"linter" yaml:"linter"`
}

type LinterConfig struct {
	Document DocumentConfig `json:"document" yaml:"document"`
}

type DocumentConfig struct {
	Entry             string            `json:"entry" yaml:"entry"`
	RequiredReachable []string          `json:"requiredReachable" yaml:"requiredReachable"`
	Excludes          []string          `json:"excludes" yaml:"excludes"`
	FrontMatter       FrontMatterConfig `json:"frontMatter" yaml:"frontMatter"`
}

type FrontMatterConfig struct {
	ExcludeFileNames []string `json:"excludeFileNames" yaml:"excludeFileNames"`
}

func Discover(root string, explicit string) (string, error) {
	if explicit != "" {
		path := explicit
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("configuration file %q does not exist", explicit)
		}
		if info.IsDir() {
			return "", fmt.Errorf("configuration path %q is a directory", explicit)
		}
		return path, nil
	}

	for _, name := range DefaultNames {
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}

	return "", fmt.Errorf("no configuration file found; expected one of %s", strings.Join(DefaultNames, ", "))
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &cfg)
	case ".json":
		err = json.Unmarshal(data, &cfg)
	case ".jsonc":
		err = json.Unmarshal(stripJSONComments(data), &cfg)
	default:
		err = fmt.Errorf("unsupported configuration extension %q", filepath.Ext(path))
	}
	if err != nil {
		return Config{}, err
	}

	if strings.TrimSpace(cfg.Linter.Document.Entry) == "" {
		return Config{}, fmt.Errorf("linter.document.entry is required")
	}

	cfg.Linter.Document.Entry = normalizeConfigPath(cfg.Linter.Document.Entry)
	cfg.Linter.Document.RequiredReachable = normalizeConfigPaths(cfg.Linter.Document.RequiredReachable)
	cfg.Linter.Document.Excludes = normalizeConfigPaths(cfg.Linter.Document.Excludes)
	cfg.Linter.Document.FrontMatter.ExcludeFileNames = normalizeFileNames(cfg.Linter.Document.FrontMatter.ExcludeFileNames)

	return cfg, nil
}

func normalizeConfigPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, item := range paths {
		item = normalizeConfigPath(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func normalizeConfigPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	path = filepath.ToSlash(path)
	return path
}

func normalizeFileNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		name = strings.ReplaceAll(name, "\\", "/")
		name = filepath.ToSlash(name)
		name = filepath.Base(name)
		if name != "." && name != "" {
			out = append(out, name)
		}
	}
	return out
}

func stripJSONComments(src []byte) []byte {
	var out bytes.Buffer
	inString := false
	escaped := false
	lineComment := false
	blockComment := false

	for i := 0; i < len(src); i++ {
		c := src[i]
		next := byte(0)
		if i+1 < len(src) {
			next = src[i+1]
		}

		if lineComment {
			if c == '\n' || c == '\r' {
				lineComment = false
				out.WriteByte(c)
			} else {
				out.WriteByte(' ')
			}
			continue
		}

		if blockComment {
			if c == '*' && next == '/' {
				out.WriteByte(' ')
				out.WriteByte(' ')
				i++
				blockComment = false
			} else if c == '\n' || c == '\r' {
				out.WriteByte(c)
			} else {
				out.WriteByte(' ')
			}
			continue
		}

		if inString {
			out.WriteByte(c)
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			out.WriteByte(c)
			continue
		}

		if c == '/' && next == '/' {
			out.WriteByte(' ')
			out.WriteByte(' ')
			i++
			lineComment = true
			continue
		}

		if c == '/' && next == '*' {
			out.WriteByte(' ')
			out.WriteByte(' ')
			i++
			blockComment = true
			continue
		}

		out.WriteByte(c)
	}

	return out.Bytes()
}

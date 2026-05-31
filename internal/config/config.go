package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var DefaultNames = []string{
	".context-lint.yaml",
	".context-lint.yml",
	".context-lint.json",
	".context-lint.jsonc",
}

const DefaultRequiredReachableMaxFileSizeBytes int64 = 32 * 1024

type Config struct {
	Linter LinterConfig `json:"linter" yaml:"linter"`
}

type LinterConfig struct {
	Document DocumentConfig `json:"document" yaml:"document"`
}

type DocumentConfig struct {
	Entry                        string            `json:"entry" yaml:"entry"`
	RequiredReachable            []string          `json:"requiredReachable" yaml:"requiredReachable"`
	RequiredReachableMaxFileSize FileSize          `json:"requiredReachableMaxFileSize" yaml:"requiredReachableMaxFileSize"`
	Excludes                     []string          `json:"excludes" yaml:"excludes"`
	FrontMatter                  FrontMatterConfig `json:"frontMatter" yaml:"frontMatter"`
}

type FrontMatterConfig struct {
	ExcludeFileNames []string `json:"excludeFileNames" yaml:"excludeFileNames"`
}

type FileSize struct {
	Bytes int64
	Set   bool
}

func (s *FileSize) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		return nil
	}
	if strings.HasPrefix(value, "\"") {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		return s.setFromString(raw)
	}
	return s.setFromString(value)
}

func (s *FileSize) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Kind == 0 || value.Tag == "!!null" {
		return nil
	}
	return s.setFromString(value.Value)
}

func (s *FileSize) EffectiveBytes(defaultBytes int64) int64 {
	if s.Set {
		return s.Bytes
	}
	return defaultBytes
}

func (s *FileSize) setFromString(value string) error {
	bytes, err := parseFileSize(value)
	if err != nil {
		return err
	}
	s.Bytes = bytes
	s.Set = true
	return nil
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

func parseFileSize(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("file size must not be empty")
	}

	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, " ", "")
	i := 0
	for i < len(value) && (value[i] >= '0' && value[i] <= '9' || value[i] == '.') {
		i++
	}
	if i == 0 {
		return 0, fmt.Errorf("file size %q must start with a number", value)
	}

	numberPart := value[:i]
	unitPart := strings.ToLower(value[i:])
	number, err := strconv.ParseFloat(numberPart, 64)
	if err != nil {
		return 0, fmt.Errorf("file size %q is not a valid number", value)
	}
	if number <= 0 || math.IsInf(number, 0) || math.IsNaN(number) {
		return 0, fmt.Errorf("file size %q must be greater than zero", value)
	}

	multiplier, ok := fileSizeMultiplier(unitPart)
	if !ok {
		return 0, fmt.Errorf("file size unit %q is not supported", unitPart)
	}
	bytes := number * float64(multiplier)
	if bytes > float64(math.MaxInt64) {
		return 0, fmt.Errorf("file size %q is too large", value)
	}
	rounded := math.Round(bytes)
	if math.Abs(bytes-rounded) > 0.0000001 {
		return 0, fmt.Errorf("file size %q must resolve to whole bytes", value)
	}
	return int64(rounded), nil
}

func fileSizeMultiplier(unit string) (int64, bool) {
	switch unit {
	case "", "b", "byte", "bytes":
		return 1, true
	case "k", "kb", "kib":
		return 1024, true
	case "m", "mb", "mib":
		return 1024 * 1024, true
	case "g", "gb", "gib":
		return 1024 * 1024 * 1024, true
	default:
		return 0, false
	}
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

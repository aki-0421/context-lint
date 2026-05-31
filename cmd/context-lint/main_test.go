package main

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestApplyBuildInfoFallbackUsesModuleVersion(t *testing.T) {
	got := applyBuildInfoFallback(buildMetadata{
		version: "dev",
		commit:  "none",
		date:    "unknown",
	}, &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
	})

	if got.version != "1.2.3" {
		t.Fatalf("version = %q, want 1.2.3", got.version)
	}
}

func TestApplyBuildInfoFallbackKeepsInjectedVersion(t *testing.T) {
	got := applyBuildInfoFallback(buildMetadata{
		version: "1.0.0",
		commit:  "abc123",
		date:    "2026-01-01T00:00:00Z",
	}, &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
	})

	if got.version != "1.0.0" {
		t.Fatalf("version = %q, want injected version 1.0.0", got.version)
	}
}

func TestApplyBuildInfoFallbackUsesVCSSettings(t *testing.T) {
	got := applyBuildInfoFallback(buildMetadata{
		version: "dev",
		commit:  "none",
		date:    "unknown",
	}, &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123"},
			{Key: "vcs.time", Value: "2026-05-24T00:00:00Z"},
		},
	})

	if got.version != "dev" {
		t.Fatalf("version = %q, want dev", got.version)
	}
	if got.commit != "abc123" {
		t.Fatalf("commit = %q, want abc123", got.commit)
	}
	if got.date != "2026-05-24T00:00:00Z" {
		t.Fatalf("date = %q, want VCS time", got.date)
	}
}

func TestSplitListArgsAllowsFlagsAfterPath(t *testing.T) {
	flagArgs, positionals := splitListArgs([]string{"docs", "--format", "json", "--strict"})

	if got, want := strings.Join(flagArgs, ","), "--format,json,--strict"; got != want {
		t.Fatalf("flagArgs = %q, want %q", got, want)
	}
	if got, want := strings.Join(positionals, ","), "docs"; got != want {
		t.Fatalf("positionals = %q, want %q", got, want)
	}
}

func TestSplitListArgsAllowsFlagsBeforePath(t *testing.T) {
	flagArgs, positionals := splitListArgs([]string{"--strict", "--root=.", "docs"})

	if got, want := strings.Join(flagArgs, ","), "--strict,--root=."; got != want {
		t.Fatalf("flagArgs = %q, want %q", got, want)
	}
	if got, want := strings.Join(positionals, ","), "docs"; got != want {
		t.Fatalf("positionals = %q, want %q", got, want)
	}
}

func TestSplitListArgsAllowsConfigAfterPath(t *testing.T) {
	flagArgs, positionals := splitListArgs([]string{"docs", "--config", ".context-lint.yaml"})

	if got, want := strings.Join(flagArgs, ","), "--config,.context-lint.yaml"; got != want {
		t.Fatalf("flagArgs = %q, want %q", got, want)
	}
	if got, want := strings.Join(positionals, ","), "docs"; got != want {
		t.Fatalf("positionals = %q, want %q", got, want)
	}
}

func TestConfigGuideTextMentionsFrontMatterExclusions(t *testing.T) {
	got := configGuideText()
	for _, want := range []string{
		"frontMatter:",
		"excludeFileNames:",
		"index.md is excluded",
		"context-lint list reports CL007",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("configGuideText() missing %q:\n%s", want, got)
		}
	}
}

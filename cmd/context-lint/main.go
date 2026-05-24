package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/aki-0421/context-lint/internal/runner"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var opts runner.Options
	var showVersion bool

	fs := flag.NewFlagSet("context-lint", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.ConfigPath, "config", "", "use a specific configuration file")
	fs.StringVar(&opts.Root, "root", "", "use a specific project root")
	fs.BoolVar(&opts.Strict, "strict", false, "treat warnings as errors")
	fs.StringVar(&opts.Format, "format", "human", "output format: human or json")
	fs.BoolVar(&opts.NoColor, "no-color", false, "disable ANSI color")
	fs.BoolVar(&showVersion, "version", false, "print version")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if showVersion {
		info := currentBuildMetadata()
		fmt.Fprintf(os.Stdout, "context-lint %s (%s, %s)\n", info.version, info.commit, info.date)
		return 0
	}

	opts.Stdout = os.Stdout
	opts.Stderr = os.Stderr
	return runner.Run(opts)
}

type buildMetadata struct {
	version string
	commit  string
	date    string
}

func currentBuildMetadata() buildMetadata {
	meta := buildMetadata{
		version: version,
		commit:  commit,
		date:    date,
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return meta
	}
	return applyBuildInfoFallback(meta, info)
}

func applyBuildInfoFallback(meta buildMetadata, info *debug.BuildInfo) buildMetadata {
	if info == nil {
		return meta
	}

	if meta.version == "" || meta.version == "dev" {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			meta.version = strings.TrimPrefix(info.Main.Version, "v")
		}
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if setting.Value != "" && (meta.commit == "" || meta.commit == "none") {
				meta.commit = setting.Value
			}
		case "vcs.time":
			if setting.Value != "" && (meta.date == "" || meta.date == "unknown") {
				meta.date = setting.Value
			}
		}
	}

	return meta
}

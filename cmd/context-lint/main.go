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
	if len(args) > 0 && args[0] == "config-guide" {
		return runConfigGuide(args[1:])
	}
	if len(args) > 0 && args[0] == "list" {
		return runList(args[1:], runner.Options{})
	}
	if len(args) > 0 && args[0] == "managed" {
		return runManaged(args[1:], runner.Options{})
	}

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
	if fs.NArg() > 0 {
		switch fs.Arg(0) {
		case "config-guide":
			return runConfigGuide(fs.Args()[1:])
		case "list":
			return runList(fs.Args()[1:], opts)
		case "managed":
			return runManaged(fs.Args()[1:], opts)
		}
		fmt.Fprintf(os.Stderr, "unknown command %q\n", fs.Arg(0))
		return 2
	}
	return runner.Run(opts)
}

func runConfigGuide(args []string) int {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "unexpected argument %q\n", args[0])
		return 2
	}
	fmt.Fprint(os.Stdout, configGuideText())
	return 0
}

func configGuideText() string {
	return `context-lint configuration guide

Front matter list exclusions

Use this when context-lint list reports CL007 for Markdown files that should not carry front matter, such as generated or routing files.

.context-lint.yaml:

linter:
  document:
    entry: AGENTS.md
    frontMatter:
      excludeFileNames:
        - README.md
        - CHANGELOG.md

Notes:
- index.md is excluded from missing-front-matter diagnostics by default.
- excludeFileNames matches file names, not paths.
- Files with front matter are still printed by context-lint list.
`
}

func runList(args []string, opts runner.Options) int {
	if opts.Format == "" {
		opts.Format = "human"
	}
	flagArgs, positionals := splitListArgs(args)

	fs := flag.NewFlagSet("context-lint list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.ConfigPath, "config", opts.ConfigPath, "use a specific configuration file")
	fs.StringVar(&opts.Root, "root", opts.Root, "use a specific project root")
	fs.BoolVar(&opts.Strict, "strict", opts.Strict, "treat warnings as errors")
	fs.StringVar(&opts.Format, "format", opts.Format, "output format: human or json")
	fs.BoolVar(&opts.NoColor, "no-color", opts.NoColor, "disable ANSI color")

	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}

	opts.Stdout = os.Stdout
	opts.Stderr = os.Stderr
	target := ""
	if len(positionals) > 0 {
		target = positionals[0]
	}
	if len(positionals) > 1 {
		fmt.Fprintf(os.Stderr, "unexpected argument %q\n", positionals[1])
		return 2
	}
	return runner.List(opts, target)
}

func runManaged(args []string, opts runner.Options) int {
	if opts.Format == "" {
		opts.Format = "human"
	}
	flagArgs, positionals := splitListArgs(args)

	fs := flag.NewFlagSet("context-lint managed", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.ConfigPath, "config", opts.ConfigPath, "use a specific configuration file")
	fs.StringVar(&opts.Root, "root", opts.Root, "use a specific project root")
	fs.BoolVar(&opts.Strict, "strict", opts.Strict, "treat warnings as errors")
	fs.StringVar(&opts.Format, "format", opts.Format, "output format: human or json")
	fs.BoolVar(&opts.NoColor, "no-color", opts.NoColor, "disable ANSI color")

	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}

	opts.Stdout = os.Stdout
	opts.Stderr = os.Stderr
	if len(positionals) == 0 {
		fmt.Fprintln(os.Stderr, "managed command is required: check, list, or tree")
		return 2
	}
	command := positionals[0]
	target := ""
	if len(positionals) > 1 {
		target = positionals[1]
	}
	if len(positionals) > 2 {
		fmt.Fprintf(os.Stderr, "unexpected argument %q\n", positionals[2])
		return 2
	}

	switch command {
	case "check":
		return runner.ManagedCheck(opts, target)
	case "list":
		return runner.ManagedList(opts, target)
	case "tree":
		return runner.ManagedTree(opts, target)
	default:
		fmt.Fprintf(os.Stderr, "unknown managed command %q\n", command)
		return 2
	}
}

func splitListArgs(args []string) ([]string, []string) {
	var flagArgs []string
	var positionals []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			flagArgs = append(flagArgs, arg)
			if listFlagNeedsValue(arg) && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}
		positionals = append(positionals, arg)
	}

	return flagArgs, positionals
}

func listFlagNeedsValue(arg string) bool {
	name := strings.TrimLeft(arg, "-")
	if _, _, hasValue := strings.Cut(name, "="); hasValue {
		return false
	}
	switch name {
	case "config", "format", "root":
		return true
	default:
		return false
	}
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

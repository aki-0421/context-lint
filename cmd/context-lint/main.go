package main

import (
	"flag"
	"fmt"
	"os"

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
		fmt.Fprintf(os.Stdout, "context-lint %s (%s, %s)\n", version, commit, date)
		return 0
	}

	opts.Stdout = os.Stdout
	opts.Stderr = os.Stderr
	return runner.Run(opts)
}

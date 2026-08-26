package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
	"github.com/RawH2O/agent-hook-kit/pkg/hookkit/app"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		if err := run(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "agent-hook-kit:", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	providerName := flags.String("provider", "", "host provider: claude or codex")
	configPath := flags.String("config", "", "explicit project config path")
	cwd := flags.String("cwd", "", "working directory override")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *providerName == "" {
		return fmt.Errorf("--provider is required")
	}

	registry := hookkit.NewRegistry()
	return app.Run(context.Background(), registry, *providerName, os.Stdin, os.Stdout, app.Options{
		ConfigPath: *configPath,
		CWD:        *cwd,
	})
}

func usage() {
	fmt.Println(`agent-hook-kit: provider-neutral hooks for Claude Code and Codex

Commands:
  run --provider claude|codex   read one hook JSON object from stdin

Project configuration is discovered from the hook cwd by walking upward:
  .agent-hook-kit.json
  .agent-hook-kit/config.json`)
}

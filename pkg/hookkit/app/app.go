// Package app provides the small executable-facing orchestration layer for
// hookkit applications. It intentionally contains no business rules.
package app

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
	"github.com/RawH2O/agent-hook-kit/pkg/hookkit/provider"
)

func mainWithArgs(args []string, rules []hookkit.Rule, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("agent-hook", flag.ContinueOnError)
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
	for _, rule := range rules {
		registry.Register(rule)
	}
	return Run(context.Background(), registry, *providerName, stdin, stdout, Options{
		ConfigPath: *configPath,
		CWD:        *cwd,
	})
}

// Options controls config discovery and the normalized working directory.
type Options struct {
	ConfigPath string
	CWD        string
}

// Run decodes one provider payload, loads the project rule selection, runs
// the rules registered by the calling application, and encodes the result.
// The runner itself deliberately has no built-in rules.
func Run(ctx context.Context, registry *hookkit.Registry, providerName string, stdin io.Reader, stdout io.Writer, options Options) error {
	if stdin == nil {
		return fmt.Errorf("hook stdin is nil")
	}
	if stdout == nil {
		return fmt.Errorf("hook stdout is nil")
	}

	adapter, err := provider.New(providerName)
	if err != nil {
		return err
	}

	raw, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read hook stdin: %w", err)
	}
	input, err := adapter.Decode(raw)
	if err != nil {
		return err
	}
	if options.CWD != "" {
		input.CWD = options.CWD
	}

	var config hookkit.Config
	if options.ConfigPath != "" {
		config, err = hookkit.LoadConfig(options.ConfigPath)
	} else {
		config, _, err = hookkit.DiscoverConfig(input.CWD)
	}
	if err != nil {
		return err
	}

	result, err := hookkit.NewRunner(registry).Run(ctx, input, config)
	if err != nil {
		return err
	}
	encoded, err := adapter.Encode(input, result)
	if err != nil {
		return err
	}
	if len(encoded) == 0 {
		return nil
	}
	if _, err := stdout.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("write hook stdout: %w", err)
	}
	return nil
}

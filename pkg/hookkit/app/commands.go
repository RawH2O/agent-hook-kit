package app

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

func runHooksCommand(command string, args []string, rules []hookkit.Rule, stdout io.Writer) error {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	providerName := flags.String("provider", "", "host provider: codex")
	hookCommand := flags.String("command", "", "complete command Codex should execute")
	outputPath := flags.String("output", "", "hooks.json path (install only; defaults to ~/.codex/hooks.json)")
	matcher := flags.String("matcher", "", "optional Codex matcher regex")
	timeout := flags.Int("timeout", 0, "optional hook timeout in seconds")
	statusMessage := flags.String("status-message", "", "optional Codex status message")
	description := flags.String("description", "", "optional hooks.json description")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.ToLower(strings.TrimSpace(*providerName)) != "codex" {
		return fmt.Errorf("--provider codex is required for %s", command)
	}
	if *timeout < 0 {
		return fmt.Errorf("--timeout cannot be negative")
	}

	if *hookCommand == "" {
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve current executable: %w", err)
		}
		*hookCommand = shellQuote(executable) + " --provider codex"
	}
	options := HookConfigOptions{
		Provider:      *providerName,
		Command:       *hookCommand,
		Matcher:       *matcher,
		Timeout:       *timeout,
		StatusMessage: *statusMessage,
		Description:   *description,
	}

	if command == "generate" {
		data, err := GenerateHooks(rules, options)
		if err != nil {
			return err
		}
		_, err = stdout.Write(data)
		return err
	}

	if *outputPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home directory: %w", err)
		}
		*outputPath = filepath.Join(home, ".codex", "hooks.json")
	}
	return InstallHooks(*outputPath, rules, options)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

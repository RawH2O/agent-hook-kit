package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

func TestRuleKeepsBusinessHandlerSmall(t *testing.T) {
	rule := Rule("test/deny", func(input Input) Result {
		if input.ToolName == "Bash" {
			return Deny("blocked")
		}
		return Allow()
	}, PreToolUse)

	result, err := rule.Run(context.Background(), hookkit.Input{ToolName: "Bash"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != hookkit.DecisionDeny || result.Reason != "blocked" {
		t.Fatalf("result = %#v", result)
	}
}

func TestMainWithArgsRegistersRulesAndUsesProjectConfig(t *testing.T) {
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, hookkit.ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"rules":["test/deny"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	rule := Rule("test/deny", func(Input) Result { return Deny("blocked") }, PreToolUse)

	input := []byte(`{"hook_event_name":"PreToolUse","cwd":"` + projectDir + `"}`)
	var output bytes.Buffer
	if err := mainWithArgs([]string{"--provider", "codex"}, []hookkit.Rule{rule}, bytes.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if output.Len() == 0 {
		t.Fatal("expected encoded denial")
	}
}

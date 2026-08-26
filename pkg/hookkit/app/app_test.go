package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

func TestRunLoadsProjectSelectionAndUsesRegisteredRule(t *testing.T) {
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, hookkit.ConfigFileName)
	if err := os.WriteFile(configPath, []byte(`{"rules":["test/deny"]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	registry := hookkit.NewRegistry().Register(hookkit.FuncRule{
		RuleID:     "test/deny",
		RuleEvents: []hookkit.Event{hookkit.EventPreToolUse},
		Fn: func(context.Context, hookkit.Input) (hookkit.Result, error) {
			return hookkit.Deny("blocked by test rule"), nil
		},
	})
	input := []byte(`{"hook_event_name":"PreToolUse","cwd":"` + projectDir + `","tool_name":"Bash","tool_input":{"command":"echo hi"}}`)
	var output bytes.Buffer

	if err := Run(context.Background(), registry, "codex", bytes.NewReader(input), &output, Options{}); err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("decode output %q: %v", output.String(), err)
	}
	hookSpecific, ok := decoded["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatalf("missing hookSpecificOutput in %#v", decoded)
	}
	if got := hookSpecific["permissionDecision"]; got != "deny" {
		t.Fatalf("permissionDecision = %v, want deny", got)
	}
}

func TestRunWithoutProjectConfigIsSilent(t *testing.T) {
	registry := hookkit.NewRegistry().Register(hookkit.FuncRule{
		RuleID:     "test/deny",
		RuleEvents: []hookkit.Event{hookkit.EventPreToolUse},
		Fn: func(context.Context, hookkit.Input) (hookkit.Result, error) {
			return hookkit.Deny("should not run"), nil
		},
	})
	input := []byte(`{"hook_event_name":"PreToolUse","cwd":"` + t.TempDir() + `"}`)
	var output bytes.Buffer

	if err := Run(context.Background(), registry, "codex", bytes.NewReader(input), &output, Options{}); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected output without config: %q", output.String())
	}
}

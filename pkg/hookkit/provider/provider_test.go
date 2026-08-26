package provider

import (
	"testing"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

func TestCodexDecodeNormalizesCommonFields(t *testing.T) {
	adapter, err := New("codex")
	if err != nil {
		t.Fatal(err)
	}
	input, err := adapter.Decode([]byte(`{
  "hook_event_name":"PreToolUse",
  "cwd":"/repo",
  "session_id":"session-1",
  "tool_name":"Bash",
  "tool_input":{"command":"git status"}
}`))
	if err != nil {
		t.Fatal(err)
	}
	if input.Event != hookkit.EventPreToolUse || input.CWD != "/repo" || input.ToolName != "Bash" {
		t.Fatalf("input = %#v", input)
	}
	if input.ToolCommand() != "git status" {
		t.Fatalf("command = %q", input.ToolCommand())
	}
}

func TestClaudeEncodeStopBlock(t *testing.T) {
	adapter, err := New("claude")
	if err != nil {
		t.Fatal(err)
	}
	data, err := adapter.Encode(hookkit.Input{
		Provider: "claude",
		Event:    hookkit.EventStop,
	}, hookkit.Block("run tests"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	wantParts := []string{`"decision":"block"`, `"reason":"run tests"`}
	for _, part := range wantParts {
		if !contains(got, part) {
			t.Fatalf("encoded %s does not contain %s", got, part)
		}
	}
}

func TestCodexEncodePreToolDenial(t *testing.T) {
	adapter, err := New("codex")
	if err != nil {
		t.Fatal(err)
	}
	data, err := adapter.Encode(hookkit.Input{
		Provider: "codex",
		Event:    hookkit.EventPreToolUse,
	}, hookkit.Deny("not allowed"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, part := range []string{`"permissionDecision":"deny"`, `"permissionDecisionReason":"not allowed"`} {
		if !contains(got, part) {
			t.Fatalf("encoded %s does not contain %s", got, part)
		}
	}
}

func TestProviderEncodeAllowIsSilent(t *testing.T) {
	adapter, err := New("codex")
	if err != nil {
		t.Fatal(err)
	}
	data, err := adapter.Encode(hookkit.Input{Event: hookkit.EventPreToolUse}, hookkit.Allow())
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("encoded allow result = %s, want empty output", data)
	}
}

func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}

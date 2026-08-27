package app

import (
	"encoding/json"
	"testing"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

func TestGenerateHookDefinitionsGroupsMultipleDefinitionsByEvent(t *testing.T) {
	data, err := GenerateHookDefinitions([]HookDefinition{
		{
			Name:    "require-mention",
			Command: "require-mention --provider codex",
			Events:  []hookkit.Event{hookkit.EventPreToolUse},
			Matcher: "Bash",
		},
		{
			Name:    "stop-summary",
			Command: "stop-summary --provider codex",
			Events:  []hookkit.Event{hookkit.EventStop},
		},
		{
			Name:    "prompt-context",
			Command: "prompt-context --provider codex",
			Events:  []hookkit.Event{hookkit.EventUserPromptSubmit, hookkit.EventStop},
			Timeout: 5,
		},
	}, "managed hooks")
	if err != nil {
		t.Fatal(err)
	}

	var document struct {
		Description string                      `json:"description"`
		Hooks       map[string][]map[string]any `json:"hooks"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document.Description != "managed hooks" {
		t.Fatalf("description = %q", document.Description)
	}
	if len(document.Hooks[string(hookkit.EventPreToolUse)]) != 1 {
		t.Fatalf("PreToolUse groups = %#v", document.Hooks[string(hookkit.EventPreToolUse)])
	}
	if len(document.Hooks[string(hookkit.EventStop)]) != 2 {
		t.Fatalf("Stop groups = %#v", document.Hooks[string(hookkit.EventStop)])
	}
	if len(document.Hooks[string(hookkit.EventUserPromptSubmit)]) != 1 {
		t.Fatalf("UserPromptSubmit groups = %#v", document.Hooks[string(hookkit.EventUserPromptSubmit)])
	}
}

func TestGenerateHookDefinitionsRejectsDuplicateNames(t *testing.T) {
	_, err := GenerateHookDefinitions([]HookDefinition{
		{Name: "same", Command: "one", Events: []hookkit.Event{hookkit.EventStop}},
		{Name: "same", Command: "two", Events: []hookkit.Event{hookkit.EventStop}},
	}, "")
	if err == nil {
		t.Fatal("expected duplicate hook name to be rejected")
	}
}

// Package provider translates Claude Code and Codex wire formats to and from
// hookkit's provider-neutral types.
package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

// Adapter owns the stdin/stdout contract of one host agent.
type Adapter interface {
	Name() string
	Decode([]byte) (hookkit.Input, error)
	Encode(hookkit.Input, hookkit.Result) ([]byte, error)
}

type Claude struct{}
type Codex struct{}

func New(name string) (Adapter, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "claude", "claude-code":
		return Claude{}, nil
	case "codex", "openai-codex":
		return Codex{}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q (want claude or codex)", name)
	}
}

func (Claude) Name() string { return "claude" }
func (Codex) Name() string  { return "codex" }

func (a Claude) Decode(data []byte) (hookkit.Input, error) {
	return decode(a.Name(), data)
}

func (a Codex) Decode(data []byte) (hookkit.Input, error) {
	return decode(a.Name(), data)
}

func (a Claude) Encode(input hookkit.Input, result hookkit.Result) ([]byte, error) {
	return encode(input, result)
}

func (a Codex) Encode(input hookkit.Input, result hookkit.Result) ([]byte, error) {
	return encode(input, result)
}

func decode(providerName string, data []byte) (hookkit.Input, error) {
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		return hookkit.Input{}, fmt.Errorf("parse %s hook stdin: %w", providerName, err)
	}
	if fields == nil {
		return hookkit.Input{}, fmt.Errorf("parse %s hook stdin: expected a JSON object", providerName)
	}

	event := firstString(fields, "hook_event_name", "hookEventName", "event", "event_name")
	if event == "" {
		return hookkit.Input{}, fmt.Errorf("parse %s hook stdin: missing hook_event_name", providerName)
	}
	cwd := firstString(fields, "cwd", "working_directory", "workdir")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	return hookkit.Input{
		Provider:             providerName,
		Event:                hookkit.Event(event),
		CWD:                  cwd,
		SessionID:            firstString(fields, "session_id", "sessionId"),
		ToolName:             firstString(fields, "tool_name", "toolName"),
		Prompt:               firstString(fields, "prompt"),
		LastAssistantMessage: firstString(fields, "last_assistant_message", "lastAssistantMessage"),
		Raw:                  append([]byte(nil), data...),
		Fields:               fields,
	}, nil
}

func firstString(fields map[string]any, names ...string) string {
	for _, name := range names {
		if value, ok := fields[name].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func encode(input hookkit.Input, result hookkit.Result) ([]byte, error) {
	if result.Decision == "" && result.Continue == nil && result.Reason == "" &&
		result.AdditionalContext == "" && result.SystemMessage == "" &&
		result.UpdatedInput == nil && !result.SuppressOutput {
		return nil, nil
	}

	output := make(map[string]any)
	if result.Continue != nil {
		output["continue"] = *result.Continue
	}
	if result.SystemMessage != "" {
		output["systemMessage"] = result.SystemMessage
	}
	if result.SuppressOutput {
		output["suppressOutput"] = true
	}

	if (result.Decision == hookkit.DecisionBlock || result.Decision == hookkit.DecisionDeny) &&
		input.Event != hookkit.EventPreToolUse && input.Event != hookkit.EventPermissionRequest {
		output["decision"] = "block"
		if result.Reason != "" {
			output["reason"] = result.Reason
		}
	}

	hookSpecific := map[string]any{
		"hookEventName": string(input.Event),
	}
	if result.AdditionalContext != "" {
		hookSpecific["additionalContext"] = result.AdditionalContext
	}

	switch input.Event {
	case hookkit.EventPreToolUse:
		if result.Decision == hookkit.DecisionDeny || result.Decision == hookkit.DecisionBlock {
			hookSpecific["permissionDecision"] = "deny"
			if result.Reason != "" {
				hookSpecific["permissionDecisionReason"] = result.Reason
			}
		}
		if result.Decision == hookkit.DecisionAllow && result.UpdatedInput != nil {
			hookSpecific["permissionDecision"] = "allow"
			hookSpecific["updatedInput"] = result.UpdatedInput
		}
	case hookkit.EventPermissionRequest:
		if result.Decision == hookkit.DecisionAllow || result.Decision == hookkit.DecisionDeny {
			decision := map[string]any{"behavior": string(result.Decision)}
			if result.Reason != "" {
				decision["message"] = result.Reason
			}
			hookSpecific["decision"] = decision
		}
	}

	if len(hookSpecific) > 1 || result.AdditionalContext != "" || input.Event == hookkit.EventPreToolUse || input.Event == hookkit.EventPermissionRequest {
		output["hookSpecificOutput"] = hookSpecific
	}
	if len(output) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("encode %s hook stdout: %w", input.Provider, err)
	}
	return data, nil
}

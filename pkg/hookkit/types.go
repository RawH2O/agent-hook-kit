// Package hookkit contains the provider-neutral hook runner primitives.
package hookkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Event is the normalized lifecycle event name shared by Claude Code and Codex.
type Event string

const (
	EventSessionStart      Event = "SessionStart"
	EventSessionEnd        Event = "SessionEnd"
	EventUserPromptSubmit  Event = "UserPromptSubmit"
	EventPreToolUse        Event = "PreToolUse"
	EventPermissionRequest Event = "PermissionRequest"
	EventPostToolUse       Event = "PostToolUse"
	EventPreCompact        Event = "PreCompact"
	EventPostCompact       Event = "PostCompact"
	EventSubagentStart     Event = "SubagentStart"
	EventSubagentStop      Event = "SubagentStop"
	EventStop              Event = "Stop"
)

// Decision describes the intent of a rule result.
//
// Block is used for lifecycle events such as Stop and UserPromptSubmit. For
// PreToolUse and PermissionRequest, Deny is the unambiguous choice.
type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
	DecisionBlock Decision = "block"
)

// Input is the normalized hook input. Raw and Fields preserve provider data so
// a rule can use provider-specific fields without making the runner provider-
// specific.
type Input struct {
	Provider             string
	Event                Event
	CWD                  string
	SessionID            string
	ToolName             string
	Prompt               string
	LastAssistantMessage string
	Raw                  json.RawMessage
	Fields               map[string]any
	Options              map[string]any
}

// Value returns a possibly nested field from the provider payload.
func (in Input) Value(path ...string) (any, bool) {
	var current any = in.Fields
	for _, part := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// StringValue returns a string field, or an empty string when the field is
// absent or has a different JSON type.
func (in Input) StringValue(path ...string) string {
	value, ok := in.Value(path...)
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return text
}

// ToolCommand extracts the common shell command field used by both providers.
func (in Input) ToolCommand() string {
	if command := in.StringValue("tool_input", "command"); command != "" {
		return command
	}
	return in.StringValue("command")
}

// Rule is an atomic, project-agnostic business rule. Project configuration
// only selects rule IDs and passes that selection's options to Run.
type Rule interface {
	ID() string
	Events() []Event
	Run(context.Context, Input) (Result, error)
}

// FuncRule is a small adapter useful for applications that prefer functions
// over defining a named type for every rule.
type FuncRule struct {
	RuleID     string
	RuleEvents []Event
	Fn         func(context.Context, Input) (Result, error)
}

func (r FuncRule) ID() string      { return r.RuleID }
func (r FuncRule) Events() []Event { return r.RuleEvents }
func (r FuncRule) Run(ctx context.Context, in Input) (Result, error) {
	if r.Fn == nil {
		return Result{}, fmt.Errorf("rule %q has no function", r.RuleID)
	}
	return r.Fn(ctx, in)
}

// Result is the provider-neutral output of a rule.
type Result struct {
	Decision          Decision
	Continue          *bool
	Reason            string
	AdditionalContext string
	SystemMessage     string
	UpdatedInput      any
	SuppressOutput    bool
}

// Allow returns an empty successful result. It intentionally produces no
// stdout, which is the least surprising hook behavior for both providers.
func Allow() Result { return Result{} }

// Deny returns a tool/permission denial result.
func Deny(reason string) Result {
	return Result{Decision: DecisionDeny, Reason: strings.TrimSpace(reason)}
}

// Block returns a lifecycle block/continuation result.
func Block(reason string) Result {
	return Result{Decision: DecisionBlock, Reason: strings.TrimSpace(reason)}
}

// Stop prevents a lifecycle event from continuing. In particular, a Stop
// hook with Continue=false allows the agent turn to finish normally.
func Stop() Result {
	continueTurn := false
	return Result{Continue: &continueTurn}
}

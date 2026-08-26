package app

import (
	"context"
	"fmt"
	"os"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

// These aliases let a hook author use only this package for ordinary rules.
type Event = hookkit.Event
type Input = hookkit.Input
type Result = hookkit.Result

type Handler func(Input) Result

const (
	SessionStart      = hookkit.EventSessionStart
	SessionEnd        = hookkit.EventSessionEnd
	UserPromptSubmit  = hookkit.EventUserPromptSubmit
	PreToolUse        = hookkit.EventPreToolUse
	PermissionRequest = hookkit.EventPermissionRequest
	PostToolUse       = hookkit.EventPostToolUse
	PreCompact        = hookkit.EventPreCompact
	PostCompact       = hookkit.EventPostCompact
	SubagentStart     = hookkit.EventSubagentStart
	SubagentStop      = hookkit.EventSubagentStop
	Stop              = hookkit.EventStop
)

// Rule adapts a business-only handler to hookkit's full Rule interface. The
// handler does not need to know about context, errors, registration, or I/O.
func Rule(id string, handler Handler, events ...Event) hookkit.Rule {
	if handler == nil {
		panic("hookkit/app: nil rule handler")
	}
	return hookkit.FuncRule{
		RuleID:     id,
		RuleEvents: events,
		Fn: func(_ context.Context, input hookkit.Input) (hookkit.Result, error) {
			return handler(input), nil
		},
	}
}

// RuleE is the escape hatch for handlers that genuinely need context or error
// propagation. Ordinary hooks should use Rule instead.
func RuleE(id string, handler func(context.Context, Input) (Result, error), events ...Event) hookkit.Rule {
	if handler == nil {
		panic("hookkit/app: nil rule handler")
	}
	return hookkit.FuncRule{
		RuleID:     id,
		RuleEvents: events,
		Fn: func(ctx context.Context, input hookkit.Input) (hookkit.Result, error) {
			return handler(ctx, input)
		},
	}
}

func Allow() Result              { return hookkit.Allow() }
func Deny(reason string) Result  { return hookkit.Deny(reason) }
func Block(reason string) Result { return hookkit.Block(reason) }
func StopTurn() Result           { return hookkit.Stop() }

// Main is the zero-boilerplate executable entry point for hook applications.
// It reads --provider/--config/--cwd, registers the supplied rules, discovers
// the project selection, and handles stdin/stdout and process errors.
func Main(rules ...hookkit.Rule) {
	if err := mainWithArgs(os.Args[1:], rules, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "agent-hook:", err)
		os.Exit(1)
	}
}

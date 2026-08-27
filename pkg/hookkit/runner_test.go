package hookkit

import (
	"context"
	"testing"
)

func TestRunnerExecutesOnlyProjectSelectedRules(t *testing.T) {
	called := map[string]int{}
	registry := NewRegistry().
		Register(FuncRule{
			RuleID:     "project-a/stop",
			RuleEvents: []Event{EventStop},
			Fn: func(_ context.Context, _ Input) (Result, error) {
				called["a"]++
				return Block("A"), nil
			},
		}).
		Register(FuncRule{
			RuleID:     "project-b/stop",
			RuleEvents: []Event{EventStop},
			Fn: func(_ context.Context, _ Input) (Result, error) {
				called["b"]++
				return Block("B"), nil
			},
		})

	result, err := NewRunner(registry).Run(context.Background(), Input{Event: EventStop}, Config{
		Rules: []RuleSelection{{ID: "project-b/stop"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called["a"] != 0 || called["b"] != 1 {
		t.Fatalf("calls = %#v, want only B once", called)
	}
	if result.Decision != DecisionBlock || result.Reason != "B" {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunnerWithoutSelectionExecutesAllRegisteredRules(t *testing.T) {
	called := map[string]int{}
	registry := NewRegistry().
		Register(FuncRule{
			RuleID:     "quality/first",
			RuleEvents: []Event{EventStop},
			Fn: func(_ context.Context, _ Input) (Result, error) {
				called["first"]++
				return Result{AdditionalContext: "first"}, nil
			},
		}).
		Register(FuncRule{
			RuleID:     "quality/second",
			RuleEvents: []Event{EventStop},
			Fn: func(_ context.Context, _ Input) (Result, error) {
				called["second"]++
				return Block("second"), nil
			},
		})

	result, err := NewRunner(registry).Run(context.Background(), Input{Event: EventStop}, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if called["first"] != 1 || called["second"] != 1 {
		t.Fatalf("calls = %#v, want every registered rule once", called)
	}
	if result.Decision != DecisionBlock || result.AdditionalContext != "first" {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunnerExplicitEmptySelectionDisablesAllRules(t *testing.T) {
	called := false
	registry := NewRegistry().Register(FuncRule{
		RuleID:     "quality/stop",
		RuleEvents: []Event{EventStop},
		Fn: func(_ context.Context, _ Input) (Result, error) {
			called = true
			return Block("unexpected"), nil
		},
	})

	result, err := NewRunner(registry).Run(context.Background(), Input{Event: EventStop}, Config{
		Rules: []RuleSelection{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("rule ran despite an explicit empty selection")
	}
	if result != (Result{}) {
		t.Fatalf("result = %#v, want empty result", result)
	}
}

func TestRunnerSkipsRuleForUnrelatedEvent(t *testing.T) {
	called := false
	registry := NewRegistry().Register(FuncRule{
		RuleID:     "quality/stop-only",
		RuleEvents: []Event{EventStop},
		Fn: func(_ context.Context, _ Input) (Result, error) {
			called = true
			return Block("unexpected"), nil
		},
	})

	result, err := NewRunner(registry).Run(context.Background(), Input{Event: EventUserPromptSubmit}, Config{
		Rules: []RuleSelection{{ID: "quality/stop-only"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("stop-only rule was called for UserPromptSubmit")
	}
	if result != (Result{}) {
		t.Fatalf("result = %#v, want empty result", result)
	}
}

func TestRunnerCombinesContextAndDenial(t *testing.T) {
	registry := NewRegistry().
		Register(FuncRule{
			RuleID:     "context/one",
			RuleEvents: []Event{EventPreToolUse},
			Fn: func(_ context.Context, _ Input) (Result, error) {
				return Result{AdditionalContext: "first"}, nil
			},
		}).
		Register(FuncRule{
			RuleID:     "safety/two",
			RuleEvents: []Event{EventPreToolUse},
			Fn: func(_ context.Context, _ Input) (Result, error) {
				return Deny("blocked"), nil
			},
		})

	result, err := NewRunner(registry).Run(context.Background(), Input{Event: EventPreToolUse}, Config{
		Rules: []RuleSelection{{ID: "context/one"}, {ID: "safety/two"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != DecisionDeny || result.Reason != "blocked" || result.AdditionalContext != "first" {
		t.Fatalf("result = %#v", result)
	}
}

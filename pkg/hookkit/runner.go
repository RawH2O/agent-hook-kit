package hookkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Runner executes only the selections in Config. Registration is the set of
// available rules; configuration is the per-project execution policy.
type Runner struct {
	Registry *Registry
}

func NewRunner(registry *Registry) *Runner {
	return &Runner{Registry: registry}
}

func (r *Runner) Run(ctx context.Context, input Input, config Config) (Result, error) {
	if r == nil || r.Registry == nil {
		return Result{}, fmt.Errorf("runner has no registry")
	}
	if err := r.Registry.Validate(); err != nil {
		return Result{}, err
	}

	var combined Result
	for index, selection := range config.Rules {
		rule, ok := r.Registry.Get(selection.ID)
		if !ok {
			return Result{}, fmt.Errorf("config rule %q is not registered", selection.ID)
		}
		if !supportsEvent(rule, input.Event) {
			continue
		}

		ruleInput := input
		ruleInput.Options = selection.Options
		result, err := rule.Run(ctx, ruleInput)
		if err != nil {
			return Result{}, fmt.Errorf("rule %q (selection %d): %w", selection.ID, index, err)
		}
		if err := mergeResult(&combined, result); err != nil {
			return Result{}, fmt.Errorf("rule %q: %w", selection.ID, err)
		}
	}
	return combined, nil
}

func mergeResult(destination *Result, source Result) error {
	if source.Decision != "" {
		destination.Decision = strongerDecision(destination.Decision, source.Decision)
	}
	if source.Continue != nil {
		if destination.Continue == nil {
			value := *source.Continue
			destination.Continue = &value
		} else if !*source.Continue {
			*destination.Continue = false
		}
	}
	destination.Reason = joinText(destination.Reason, source.Reason)
	destination.AdditionalContext = joinText(destination.AdditionalContext, source.AdditionalContext)
	destination.SystemMessage = joinText(destination.SystemMessage, source.SystemMessage)
	destination.SuppressOutput = destination.SuppressOutput || source.SuppressOutput

	if source.UpdatedInput != nil {
		if destination.UpdatedInput == nil {
			destination.UpdatedInput = source.UpdatedInput
		} else {
			left, leftErr := json.Marshal(destination.UpdatedInput)
			right, rightErr := json.Marshal(source.UpdatedInput)
			if leftErr != nil || rightErr != nil || !bytes.Equal(left, right) {
				return fmt.Errorf("multiple rules produced conflicting updatedInput values")
			}
		}
	}
	return nil
}

func strongerDecision(left, right Decision) Decision {
	rank := map[Decision]int{DecisionAllow: 1, DecisionBlock: 2, DecisionDeny: 3}
	if rank[right] > rank[left] {
		return right
	}
	return left
}

func joinText(left, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	switch {
	case left == "":
		return right
	case right == "":
		return left
	default:
		return left + "\n" + right
	}
}

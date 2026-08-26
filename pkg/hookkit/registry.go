package hookkit

import (
	"fmt"
	"sort"
)

// Registry stores all rules compiled into a runner. It does not decide which
// rules execute; that decision belongs to the project config.
type Registry struct {
	rules map[string]Rule
	order []string
}

func NewRegistry() *Registry {
	return &Registry{rules: make(map[string]Rule)}
}

// Register adds a rule and returns the registry for fluent setup.
func (r *Registry) Register(rule Rule) *Registry {
	if rule == nil || rule.ID() == "" {
		panic("hookkit: cannot register a nil rule or a rule without an ID")
	}
	if _, exists := r.rules[rule.ID()]; exists {
		panic("hookkit: duplicate rule ID " + rule.ID())
	}
	r.rules[rule.ID()] = rule
	r.order = append(r.order, rule.ID())
	return r
}

func (r *Registry) Get(id string) (Rule, bool) {
	rule, ok := r.rules[id]
	return rule, ok
}

// IDs returns registered IDs in registration order. It is intended for CLI
// discovery and documentation, not for execution ordering.
func (r *Registry) IDs() []string {
	ids := append([]string(nil), r.order...)
	return ids
}

// Validate checks the registry's own invariants without executing rules.
func (r *Registry) Validate() error {
	if r == nil {
		return fmt.Errorf("nil registry")
	}
	for id, rule := range r.rules {
		if id == "" || rule == nil {
			return fmt.Errorf("invalid rule registration %q", id)
		}
		if len(rule.Events()) == 0 {
			return fmt.Errorf("rule %q does not declare any events", id)
		}
	}
	return nil
}

// SortedIDs is useful when deterministic output independent of registration
// order is preferred.
func (r *Registry) SortedIDs() []string {
	ids := r.IDs()
	sort.Strings(ids)
	return ids
}

func supportsEvent(rule Rule, event Event) bool {
	for _, supported := range rule.Events() {
		if supported == event || supported == "*" {
			return true
		}
	}
	return false
}

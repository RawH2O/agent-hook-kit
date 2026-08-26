package rules

import (
	"context"
	"os/exec"
	"regexp"
	"strings"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

// RegisterBuiltins adds deliberately small examples. Real applications can
// register their own rules without changing the runner or provider adapters.
func RegisterBuiltins(registry *hookkit.Registry) {
	registry.Register(DangerousShell{})
	registry.Register(PromptContext{})
	registry.Register(CleanWorktreeOnStop{})
	registry.Register(CommentMentionRequired{})
}

type DangerousShell struct{}

func (DangerousShell) ID() string              { return "safety/no-dangerous-shell" }
func (DangerousShell) Events() []hookkit.Event { return []hookkit.Event{hookkit.EventPreToolUse} }
func (DangerousShell) Run(_ context.Context, input hookkit.Input) (hookkit.Result, error) {
	if input.ToolName != "" && !strings.EqualFold(input.ToolName, "bash") && !strings.EqualFold(input.ToolName, "shell") {
		return hookkit.Allow(), nil
	}
	command := input.ToolCommand()
	if command == "" || !dangerousShellPattern.MatchString(command) {
		return hookkit.Allow(), nil
	}
	return hookkit.Deny("命令命中默认危险操作规则，请确认后再执行：" + command), nil
}

var dangerousShellPattern = regexp.MustCompile(`(?i)(rm\s+-[a-z]*r[a-z]*f|git\s+reset\s+--hard|git\s+clean\s+-[a-z]*f)`)

type PromptContext struct{}

func (PromptContext) ID() string              { return "context/prompt" }
func (PromptContext) Events() []hookkit.Event { return []hookkit.Event{hookkit.EventUserPromptSubmit} }
func (PromptContext) Run(_ context.Context, input hookkit.Input) (hookkit.Result, error) {
	contextText, _ := input.Options["additional_context"].(string)
	if strings.TrimSpace(contextText) == "" {
		return hookkit.Allow(), nil
	}
	return hookkit.Result{AdditionalContext: contextText}, nil
}

type CleanWorktreeOnStop struct{}

func (CleanWorktreeOnStop) ID() string              { return "git/clean-worktree-on-stop" }
func (CleanWorktreeOnStop) Events() []hookkit.Event { return []hookkit.Event{hookkit.EventStop} }
func (CleanWorktreeOnStop) Run(ctx context.Context, input hookkit.Input) (hookkit.Result, error) {
	if input.CWD == "" {
		return hookkit.Allow(), nil
	}
	command := exec.CommandContext(ctx, "git", "-C", input.CWD, "status", "--porcelain")
	output, err := command.Output()
	if err != nil {
		// Non-git directories should not make Stop unusable. A project can add a
		// stricter custom rule if git status must be mandatory.
		return hookkit.Allow(), nil
	}
	if len(strings.TrimSpace(string(output))) == 0 {
		return hookkit.Allow(), nil
	}
	return hookkit.Block("工作区还有未提交变更，请先检查并完成收尾：\n" + strings.TrimSpace(string(output))), nil
}

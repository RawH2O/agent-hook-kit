package rules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

func TestCommentMentionRequired(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		wantDenied bool
	}{
		{
			name:    "agent mention",
			command: `multica issue comment add MUL-1 --content '请处理 [@Reviewer](mention://agent/7f3a0000-0000-0000-0000-000000000001)'`,
		},
		{
			name:    "member mention",
			command: `multica issue comment add MUL-1 --content '请确认 [@Alice](mention://member/7f3a0000-0000-0000-0000-000000000002)'`,
		},
		{
			name:    "squad mention",
			command: `multica issue comment add MUL-1 --content '请接手 [@Review](mention://squad/7f3a0000-0000-0000-0000-000000000003)'`,
		},
		{
			name:       "plain at name",
			command:    `multica issue comment add MUL-1 --content '@Alice 请看一下'`,
			wantDenied: true,
		},
		{
			name:       "all mention",
			command:    `multica issue comment add MUL-1 --content '通知 [@all](mention://all/all)'`,
			wantDenied: true,
		},
		{
			name:       "issue mention",
			command:    `multica issue comment add MUL-1 --content '关联 [@Issue](mention://issue/7f3a0000-0000-0000-0000-000000000004)'`,
			wantDenied: true,
		},
		{
			name:       "content stdin",
			command:    `multica issue comment add MUL-1 --content-stdin`,
			wantDenied: true,
		},
		{
			name:    "environment assignment",
			command: `MULTICA_PROFILE=work multica issue comment add MUL-1 --content '请复核 [@Reviewer](mention://agent/7f3a0000-0000-0000-0000-000000000005)'`,
		},
		{
			name:    "unrelated command",
			command: `multica issue get MUL-1`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := (CommentMentionRequired{}).Run(context.Background(), commandInput(test.command, ""))
			if err != nil {
				t.Fatal(err)
			}
			denied := result.Decision == hookkit.DecisionDeny
			if denied != test.wantDenied {
				t.Fatalf("denied = %v, result = %#v", denied, result)
			}
		})
	}
}

func TestCommentMentionRequiredReadsContentFileRelativeToCWD(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "comment.md"), []byte("请复核 [@Reviewer](mention://agent/7f3a0000-0000-0000-0000-000000000006)"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := (CommentMentionRequired{}).Run(context.Background(), commandInput("multica issue comment add MUL-1 --content-file comment.md", directory))
	if err != nil {
		t.Fatal(err)
	}
	if result != (hookkit.Result{}) {
		t.Fatalf("result = %#v, want allow", result)
	}
}

func TestCommentMentionRequiredIgnoresNonShellTools(t *testing.T) {
	input := commandInput("multica issue comment add MUL-1 --content 'no mention'", "")
	input.ToolName = "mcp__multica__issue_comment_add"
	result, err := (CommentMentionRequired{}).Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result != (hookkit.Result{}) {
		t.Fatalf("result = %#v, want allow", result)
	}
}

func commandInput(command, cwd string) hookkit.Input {
	return hookkit.Input{
		Event:    hookkit.EventPreToolUse,
		CWD:      cwd,
		ToolName: "Bash",
		Fields: map[string]any{
			"tool_input": map[string]any{"command": command},
		},
	}
}

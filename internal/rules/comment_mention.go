package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/RawH2O/agent-hook-kit/pkg/hookkit"
)

// CommentMentionRequired requires Multica comments to address a concrete
// agent, squad, or human. The provider adapter has already normalized the
// hook payload; this rule only inspects the command's business arguments.
type CommentMentionRequired struct{}

func (CommentMentionRequired) ID() string {
	return "multica/comment-mention-required"
}

func (CommentMentionRequired) Events() []hookkit.Event {
	return []hookkit.Event{hookkit.EventPreToolUse}
}

func (CommentMentionRequired) Run(_ context.Context, input hookkit.Input) (hookkit.Result, error) {
	if !isShellTool(input.ToolName) {
		return hookkit.Allow(), nil
	}
	command := input.ToolCommand()
	if strings.TrimSpace(command) == "" {
		return hookkit.Allow(), nil
	}

	words, err := splitShellWords(command)
	if err != nil {
		if strings.Contains(command, "multica") {
			return hookkit.Deny(fmt.Sprintf("无法解析 multica 评论命令，已阻止执行：%v", err)), nil
		}
		return hookkit.Allow(), nil
	}
	args, ok := multicaCommentArgs(words)
	if !ok {
		return hookkit.Allow(), nil
	}

	content, found, reason := readCommentContent(args, input.CWD)
	if reason != "" {
		return hookkit.Deny(reason), nil
	}
	if !found {
		return hookkit.Deny("multica 评论必须提供 --content 或 --content-file，并且内容必须 @ 一名其他 agent 或人类。"), nil
	}
	if !mentionPattern.MatchString(content) {
		return hookkit.Deny("multica 评论必须包含真实的 agent、人类或 squad mention，例如 [@Reviewer](mention://agent/<uuid>)；普通 @名字、@all 或 issue mention 不算。"), nil
	}
	return hookkit.Allow(), nil
}

var mentionPattern = regexp.MustCompile(`\[[^\]\r\n]*\]\(mention://(?:agent|squad|member)/[0-9a-fA-F-]+\)`)

func isShellTool(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return name == "" || name == "bash" || name == "shell" || name == "sh" || name == "exec_command"
}

func multicaCommentArgs(words []string) ([]string, bool) {
	index := findMulticaExecutable(words)
	if index < 0 {
		return nil, false
	}
	args := words[index+1:]
	if len(args) < 3 || args[0] != "issue" || args[1] != "comment" || args[2] != "add" {
		return nil, false
	}
	return args[3:], true
}

func findMulticaExecutable(words []string) int {
	index := 0
	for index < len(words) && isEnvironmentAssignment(words[index]) {
		index++
	}
	if index >= len(words) {
		return -1
	}
	if strings.EqualFold(filepath.Base(words[index]), "multica") || strings.EqualFold(filepath.Base(words[index]), "multica.exe") {
		return index
	}
	if !strings.EqualFold(filepath.Base(words[index]), "env") {
		return -1
	}

	index++
	for index < len(words) {
		word := words[index]
		if word == "--" {
			index++
			break
		}
		if isEnvironmentAssignment(word) || strings.HasPrefix(word, "-") {
			index++
			continue
		}
		break
	}
	if index < len(words) && (strings.EqualFold(filepath.Base(words[index]), "multica") || strings.EqualFold(filepath.Base(words[index]), "multica.exe")) {
		return index
	}
	return -1
}

func isEnvironmentAssignment(word string) bool {
	if word == "" {
		return false
	}
	for index, character := range word {
		if character == '=' {
			return index > 0
		}
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && character != '_' && (index == 0 || character < '0' || character > '9') {
			return false
		}
	}
	return false
}

func readCommentContent(args []string, cwd string) (string, bool, string) {
	var content string
	found := false
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--content-stdin":
			return "", false, "无法校验 --content-stdin 的评论内容；请改用 --content 或 --content-file，并在评论中加入 agent/人类 mention。"
		case argument == "--content":
			if index+1 >= len(args) {
				return "", false, "--content 缺少评论正文；评论必须 @ 一名其他 agent 或人类。"
			}
			content = args[index+1]
			found = true
			index++
		case strings.HasPrefix(argument, "--content="):
			content = strings.TrimPrefix(argument, "--content=")
			found = true
		case argument == "--content-file":
			if index+1 >= len(args) {
				return "", false, "--content-file 缺少文件路径；评论必须 @ 一名其他 agent 或人类。"
			}
			content, found = readContentFile(args[index+1], cwd)
			if !found {
				return "", false, fmt.Sprintf("无法读取评论文件 %s", resolveContentPath(args[index+1], cwd))
			}
			index++
		case strings.HasPrefix(argument, "--content-file="):
			path := strings.TrimPrefix(argument, "--content-file=")
			content, found = readContentFile(path, cwd)
			if !found {
				return "", false, fmt.Sprintf("无法读取评论文件 %s", resolveContentPath(path, cwd))
			}
		}
	}
	return content, found, ""
}

func readContentFile(value, cwd string) (string, bool) {
	data, err := os.ReadFile(resolveContentPath(value, cwd))
	if err != nil {
		return "", false
	}
	return string(data), true
}

func resolveContentPath(value, cwd string) string {
	path := filepath.Clean(value)
	if filepath.IsAbs(path) || cwd == "" {
		return path
	}
	return filepath.Join(cwd, path)
}

// splitShellWords handles the quoting needed for a command hook without
// invoking a shell. It is intentionally a tokenizer, not a shell evaluator:
// variables and command substitutions remain literal and therefore fail
// closed when they are used as the comment source.
func splitShellWords(command string) ([]string, error) {
	var words []string
	var word strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	started := false

	flush := func() {
		if started {
			words = append(words, word.String())
			word.Reset()
			started = false
		}
	}

	for _, character := range command {
		if escaped {
			word.WriteRune(character)
			escaped = false
			started = true
			continue
		}
		if inSingleQuote {
			if character == '\'' {
				inSingleQuote = false
			} else {
				word.WriteRune(character)
			}
			started = true
			continue
		}
		if inDoubleQuote {
			switch character {
			case '"':
				inDoubleQuote = false
			case '\\':
				escaped = true
			default:
				word.WriteRune(character)
			}
			started = true
			continue
		}

		switch {
		case character == '\'':
			inSingleQuote = true
			started = true
		case character == '"':
			inDoubleQuote = true
			started = true
		case character == '\\':
			escaped = true
			started = true
		case unicode.IsSpace(character):
			flush()
		default:
			word.WriteRune(character)
			started = true
		}
	}

	if escaped || inSingleQuote || inDoubleQuote {
		return nil, fmt.Errorf("命令包含未闭合的引号或转义")
	}
	flush()
	return words, nil
}

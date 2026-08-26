# agent-hook-kit

一个用 Go 编写、同时适配 Claude Code 和 Codex 的 Hook Runner。

它解决的核心问题是把三件事拆开：

1. provider adapter 负责 stdin JSON、事件名和 stdout JSON 的协议差异；
2. `hookkit.Rule` 只实现一条与项目无关的业务规则；
3. 项目根目录的 `.agent-hook-kit.json` 只选择当前项目启用哪些规则。

因此，同一个二进制可以被多个项目复用，而项目 A 和项目 B 只需要各自维护自己的规则选择，不需要在规则代码里写路径判断，也不需要在 Claude/Codex 配置中逐条复制规则。

## 当前 MVP

```bash
go build ./cmd/agent-hook-kit
./agent-hook-kit list-rules
```

内置示例规则：

- `safety/no-dangerous-shell`：拦截一组明显危险的 Bash 命令；
- `context/prompt`：在 `UserPromptSubmit` 注入项目配置里的额外上下文；
- `git/clean-worktree-on-stop`：在 `Stop` 时发现工作区有变更就请求继续收尾。

没有找到项目配置时，runner 不执行任何规则并静默放行。这是有意设计：避免“注册进二进制的所有规则默认对所有项目生效”。

## 项目配置

在每个项目根目录放置 `.agent-hook-kit.json`：

```json
{
  "rules": [
    "safety/no-dangerous-shell",
    {
      "id": "context/prompt",
      "options": {
        "additional_context": "本项目修改后必须运行 go test ./...。"
      }
    }
  ]
}
```

Runner 会从 hook 输入里的 `cwd` 开始向上查找配置，所以不依赖任何人的绝对目录。也支持 `.agent-hook-kit/config.json`。

## Claude Code 配置

在 Claude Code 的 hook 配置中只配置一个统一入口：

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "agent-hook-kit run --provider claude"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "agent-hook-kit run --provider claude"
          }
        ]
      }
    ]
  }
}
```

## Codex 配置

Codex 使用同一个 runner，只替换 provider：

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "agent-hook-kit run --provider codex"
          }
        ]
      }
    ]
  }
}
```

`PreToolUse`、`PermissionRequest`、`PostToolUse` 等事件也使用同一个入口；规则通过 `Events()` 声明自己关心的事件。

## 编写自己的规则

规则作者不需要知道谁调用它，也不需要写项目分流：

```go
registry := hookkit.NewRegistry().Register(hookkit.FuncRule{
    RuleID:     "quality/require-tests",
    RuleEvents: []hookkit.Event{hookkit.EventStop},
    Fn: func(ctx context.Context, in hookkit.Input) (hookkit.Result, error) {
        // 这里只写规则本身；in.CWD 是宿主传入的当前项目目录。
        return hookkit.Allow(), nil
    },
})
runner := hookkit.NewRunner(registry)
```

把这个规则编译进你的 runner 后，各项目只在 `.agent-hook-kit.json` 中填写 `quality/require-tests` 即可。

## 设计边界

当前版本先统一最重要的 stdin/stdout、事件归一化、规则注册、项目配置查找和结果合并。Codex 与 Claude 的更细粒度 matcher、超时、异步 hook、配置安装命令将在协议适配稳定后继续补齐。

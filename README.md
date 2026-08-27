# agent-hook-kit

一个用 Go 编写、同时适配 Claude Code 和 Codex 的 provider-neutral Hook Runner 基础库。

这个仓库只负责通用基础设施：

- provider adapter：解析 Claude Code/Codex 的 stdin，并编码 hook stdout；
- `hookkit.Input`、`hookkit.Result` 等跨 provider 类型；
- `Registry`、项目配置发现和按项目选择规则的 `Runner`；
- `app.Run`：把 stdin、配置、Runner 和 stdout 串起来。

基础库不内置任何业务规则，也不包含 Multica、Git、安全策略或某个具体项目的代码。业务项目应在自己的仓库里定义规则、注册规则，并通过 `.agent-hook-kit.json` 选择当前项目要启用的规则。

## 快速使用

业务项目只需要写业务 handler，再用一行声明规则：

```go
func check(input app.Input) app.Result {
    // 这里只实现一条规则，不判断项目名称或绝对路径。
    return app.Allow()
}

func main() {
    app.Main(app.Rule("quality/require-tests", check, app.Stop))
}
```

项目根目录的 `.agent-hook-kit.json` 只负责选择规则：

```json
{
  "rules": ["quality/require-tests"]
}
```

Runner 从 hook 输入中的 `cwd` 开始向上查找配置，也支持 `.agent-hook-kit/config.json`。没有配置时默认执行所有已注册且匹配当前事件的规则；如果项目需要裁剪规则，再写配置文件。显式的 `"rules": []` 表示该项目禁用全部规则。

## 可执行入口

仓库中的 `cmd/agent-hook-kit` 是一个不带业务规则的通用 smoke-test 入口：

```bash
go run ./cmd/agent-hook-kit --provider claude
go run ./cmd/agent-hook-kit --provider codex
```

实际业务应用通常只需要调用 `app.Main`。它会自动处理参数、规则注册、配置发现、stdin/stdout 和错误退出；Claude Code 和 Codex 的配置都只需要指向这个业务应用的统一入口。

平台或 GitOps 层如果已经有独立的 Hook 元数据，也可以直接生成 Codex 配置，不需要导入业务规则：

```go
data, err := app.GenerateHookDefinitions([]app.HookDefinition{
    {
        Name: "require-mention",
        Command: "multica-comment-mention-required --provider codex",
        Events: []app.Event{app.PreToolUse},
        Matcher: "Bash",
    },
}, "Managed by Multica")
```

这里的 `HookDefinition` 就是 YAML 清单到 `hooks.json` 的中间层；项目名、项目绝对路径和业务规则代码都不进入基础库。

## 编写规则

规则实现只依赖 provider-neutral 的 `hookkit.Input` 和 `hookkit.Result`：

```go
func requireTests(input app.Input) app.Result {
    return app.Allow()
}
```

`app.Rule` 会把 handler 自动包装成规则。需要 context 或 error 传播时才使用 `app.RuleE`；需要自定义 I/O 或配置时才直接使用底层 `app.Run`。注册是可用规则集合；配置是可选的裁剪策略。没有配置时，已注册且匹配当前事件的规则全部执行；配置存在时，只有其中列出的规则执行。

## 当前边界

当前版本统一 stdin/stdout、事件归一化、规则注册、项目配置查找和结果合并。更细粒度的 provider matcher、超时、异步 hook 以及安装命令，会在协议适配稳定后继续补齐。

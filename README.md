# agent-hook-kit

一个用 Go 编写、同时适配 Claude Code 和 Codex 的 provider-neutral Hook Runner 基础库。

这个仓库只负责通用基础设施：

- provider adapter：解析 Claude Code/Codex 的 stdin，并编码 hook stdout；
- `hookkit.Input`、`hookkit.Result` 等跨 provider 类型；
- `Registry`、项目配置发现和按项目选择规则的 `Runner`；
- `app.Run`：把 stdin、配置、Runner 和 stdout 串起来。

基础库不内置任何业务规则，也不包含 Multica、Git、安全策略或某个具体项目的代码。业务项目应在自己的仓库里定义规则、注册规则，并通过 `.agent-hook-kit.json` 选择当前项目要启用的规则。

## 快速使用

业务项目定义并注册自己的规则：

```go
registry := hookkit.NewRegistry().Register(hookkit.FuncRule{
    RuleID:     "quality/require-tests",
    RuleEvents: []hookkit.Event{hookkit.EventStop},
    Fn: func(ctx context.Context, in hookkit.Input) (hookkit.Result, error) {
        // 这里只实现一条规则，不判断项目名称或绝对路径。
        return hookkit.Allow(), nil
    },
})

err := app.Run(ctx, registry, "claude", os.Stdin, os.Stdout, app.Options{})
```

项目根目录的 `.agent-hook-kit.json` 只负责选择规则：

```json
{
  "rules": ["quality/require-tests"]
}
```

Runner 从 hook 输入中的 `cwd` 开始向上查找配置，也支持 `.agent-hook-kit/config.json`。没有配置时不会执行任何已注册规则，并静默放行。

## 可执行入口

仓库中的 `cmd/agent-hook-kit` 是一个不带业务规则的通用 smoke-test 入口：

```bash
go run ./cmd/agent-hook-kit run --provider claude
go run ./cmd/agent-hook-kit run --provider codex
```

实际业务应用通常应提供自己的 `main`，注册自己的规则后再调用 `app.Run`。Claude Code 和 Codex 的配置都只需要指向这个业务应用的统一入口；具体项目通过 `.agent-hook-kit.json` 选择规则。

## 编写规则

规则实现只依赖 provider-neutral 的 `hookkit.Input` 和 `hookkit.Result`：

```go
type RequireTests struct{}

func (RequireTests) ID() string { return "quality/require-tests" }

func (RequireTests) Events() []hookkit.Event {
    return []hookkit.Event{hookkit.EventStop}
}

func (RequireTests) Run(ctx context.Context, in hookkit.Input) (hookkit.Result, error) {
    return hookkit.Allow(), nil
}
```

配置是执行策略，注册是可用规则集合。只有同时满足“规则已注册”和“规则 ID 出现在项目配置中”时，规则才会执行。

## 当前边界

当前版本统一 stdin/stdout、事件归一化、规则注册、项目配置查找和结果合并。更细粒度的 provider matcher、超时、异步 hook 以及安装命令，会在协议适配稳定后继续补齐。

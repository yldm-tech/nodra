# Nodra

Nodra 是面向 Go 桌面应用的并发安全响应式状态库，为 [MyGo](https://github.com/egoist/mygo) 提供类似 Zustand 的使用体验，同时让状态与渲染器和持久化后端保持独立。

[English](README.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md)

## 安装

```sh
go get github.com/yldm-tech/nodra@latest
```

核心包是 `github.com/yldm-tech/nodra`，可选的本地检查器是 `github.com/yldm-tech/nodra/devtools`。

## 快速开始

```go
store := nodra.New(State{})
defer store.Close()
stop := store.Subscribe(func(s nodra.Snapshot[State]) { fmt.Println(s.Value) })
defer stop()
_ = store.Update(func(s *State) { s.Count++ })
```

## 核心能力

- 带版本号的 `Store[T]`、同步订阅、有限 Watch 和只读视图。
- 派生状态、选择器、投影、字段绑定和原子事务。
- 支持分组、过滤和大小预算的撤销/重做历史。
- JSON、版本迁移、加密、备份、日志恢复和冲突检测持久化。
- 支持防抖、取消、排队、进度、重试和指标的异步 Action。
- 用 `Scope` 管理资源所有权和关闭顺序。
- MyGo 刷新绑定和本地认证 DevTools 检查器。

## MyGo

```go
stop := nodra.InvalidateSelector(store, window, func(s State) int { return s.Count })
defer stop()
```

MyGo 集成目标版本是 `v0.6.3`。临时控件状态应放在视图所有者中，共享业务状态只提交实际编辑的字段。参见 [`examples/mygo`](examples/mygo/main.go)。

## 完整示例

[`examples/workbench`](examples/workbench/README.md) 是完整的 Nodra + MyGo 桌面项目，包含克隆状态、本地草稿、带校验和备份的持久化、可防抖和取消的异步搜索、异步保存、Scope 关闭、原生刷新、英简双语和可选 DevTools。

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
```

## DevTools 和文档

DevTools 默认只读，设置 `AllowRestore` 才会启用类型安全的 JSON 编辑和快照恢复。详见 [`docs/devtools.md`](docs/devtools.md)。

- [`docs/public-api.md`](docs/public-api.md)：公共 API 索引。
- [`docs/architecture.md`](docs/architecture.md)：所有权、并发和生命周期语义。
- [`docs/compatibility.md`](docs/compatibility.md)：v0.x 到 v1 兼容策略。
- [`docs/support.md`](docs/support.md)：Go、MyGo、操作系统和架构支持矩阵。

## 验证

```sh
go test ./...
go test -race ./...
go vet ./...
```

项目要求 Go `1.27.3+`，原生集成使用 MyGo `v0.6.3`。

# Nodra

[![Website](https://img.shields.io/badge/website-nodra.yldm.ai-c8f05a)](https://nodra.yldm.ai)
[![Go Reference](https://pkg.go.dev/badge/github.com/yldm-tech/nodra.svg)](https://pkg.go.dev/github.com/yldm-tech/nodra)
[![Latest version](https://img.shields.io/github/v/release/yldm-tech/nodra)](https://github.com/yldm-tech/nodra/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/yldm-tech/nodra)](go.mod)
[![GitHub stars](https://img.shields.io/github/stars/yldm-tech/nodra?style=flat)](https://github.com/yldm-tech/nodra/stargazers)
[![License](https://img.shields.io/github/license/yldm-tech/nodra)](LICENSE)
[![X](https://img.shields.io/badge/X-%40suzukaze__haduki-000000?logo=x&logoColor=white)](https://x.com/suzukaze_haduki)
[![GitHub Sponsors](https://img.shields.io/badge/GitHub%20Sponsors-ea4aaa?logo=githubsponsors&logoColor=white)](https://github.com/sponsors/houko)

Nodra 是面向 Go 桌面应用的并发安全响应式状态库，为 [MyGo](https://github.com/egoist/mygo) 提供类似 Zustand 的使用体验，同时让状态与渲染器和持久化后端保持独立。

[English](README.md) · [日本語](README.ja.md) · [한국어](README.ko.md) · [Español](README.es.md)

## 安装

```sh
go get github.com/yldm-tech/nodra@latest
```

核心包是 `github.com/yldm-tech/nodra`。可选的本地检查器是 `github.com/yldm-tech/nodra/devtools`，可选的网页桥接是 `github.com/yldm-tech/nodra/bridge`，原生 UI 的 Edit 菜单集成是 `github.com/yldm-tech/nodra/nativeui`。

## 快速开始

```go
package main

import (
	"fmt"

	"github.com/yldm-tech/nodra"
)

type State struct{ Count int }

func main() {
	store := nodra.New(State{})
	defer store.Close()

	stop := store.Subscribe(func(s nodra.Snapshot[State]) {
		fmt.Println(s.Value.Count)
	})
	defer stop()

	_ = store.Update(func(s *State) { s.Count++ })
}
```

## 核心能力

- 带版本号的 `Store[T]`、同步订阅、有限 Watch 和只读视图。
- 派生状态、选择器、投影、字段绑定和原子事务。
- 支持分组、过滤和大小预算的撤销/重做历史。
- JSON、版本迁移、加密、备份、日志恢复和冲突检测持久化。
- 支持防抖、取消、排队、进度、重试和指标的异步 Action。
- 用 `Scope` 管理资源所有权和关闭顺序。
- 可选的 MyGo 刷新辅助函数和本地认证 DevTools 检查器。
- 网页桥接：MyGo 应用的网页可以跟随 Store，并通过注册的 Action 修改它。
- 原生 UI 窗口的撤销/重做快捷键和 Edit 菜单作用于 History。

## MyGo

```go
mygo.App.WhenReady(func() {
	window := mygo.NewWindow(mygo.WindowOptions{Title: "Counter", Content: ui.View(view)})
	// Redraws stop when the window closes. Keep the returned stop only to end them sooner: a defer here would end them as soon as WhenReady returns.
	nodra.InvalidateSelector(store, window, func(s State) int {
		return s.Count
	})
})
```

MyGo 集成目标版本是 `v0.7.4`。临时控件状态应放在视图所有者中，共享 Store 只提交实际编辑的字段。最小的原生窗口示例见 [`examples/mygo`](examples/mygo/main.go)。

```go
history := nodra.Track(store, 100)
nativeui.InvalidateHistory(history, window) // redraw when Undo or Redo becomes possible

func (app *App) view(c *ui.Context) {
	nativeui.UndoShortcuts(c, app.history) // Cmd+Z, Shift+Cmd+Z and the Edit menu
	// ...
}
```

可选的 `nativeui` 子包让窗口的撤销和重做作用于 `History`：快捷键，以及 MyGo 0.7.4 起 macOS 上 Edit 菜单里的对应项，只在历史可以撤销或重做时启用。聚焦的文本框保留自己的撤销。它独立于核心包，没有原生 UI 的应用不会因此链接 MyGo 的 `ui` 包。

只共享 Store、从不打开窗口的服务和命令行工具可以用 `-tags nodra_nomygo` 构建。它去掉接收 `*mygo.Window` 的辅助函数（`InvalidateTarget` 保留），因此不会链接 MyGo，`CGO_ENABLED=0` 的 Linux 构建就是静态可执行文件。这个标签作用于整个构建：`nativeui` 以及任何调用这些辅助函数的库在该标签下无法编译，`bridge` 会重新链接 MyGo。详见 [`docs/support.md`](docs/support.md)。

## 网页

```go
b := bridge.New(bridge.Options{})
stop, err := bridge.ExposeProjection(b, "counter", store, func(s State) PageState { return PageState{Count: s.Count} })
stopAdd, err := bridge.Handle(b, "counter/add", func(ctx context.Context, by int) error {
	return store.Update(func(s *State) { s.Count += by })
})
b.Bind() // before App.Run
```

```js
import { dispatch, remoteState } from "./nodra.js";

const counter = remoteState("counter"); // subscribe and getSnapshot fit useSyncExternalStore
await dispatch("counter/add", 1);
```

可选的 `bridge` 子包让一个 Store 同时作为原生窗口和网页的唯一状态源。网页只看到投影返回的值，值变化时才收到新帧（网页处理慢、积压约 1 MiB 未取走的帧后才合并为最新一帧），也只能通过 Go 注册的 Action 修改状态。[`bridge/client`](bridge/client/nodra.js) 里的 JavaScript 客户端没有任何依赖。详见 [`docs/bridge.md`](docs/bridge.md) 和 [`examples/web`](examples/web/main.go)。

## 持久化

```go
persistence := nodra.NewPersistence(store, nodra.FileStorage{Path: "state.json", Backup: true}, nodra.JSONCodec[State]{})
// A missing file is a first start. Stop on any other load error: autosave would overwrite the file.
if err := persistence.Load(ctx); err != nil && !errors.Is(err, fs.ErrNotExist) {
	return err
}
writer, err := persistence.AutoSaveAsync(ctx, nodra.AsyncOptions{Debounce: 100 * time.Millisecond})
if err != nil {
	return err
}
defer func() {
	if err := writer.Close(shutdownCtx); err != nil {
		// The store still owns the data: retry with persistence.Save using a fresh context, or report the error.
		log.Printf("save state: %v", err)
	}
}()
```

应用需要迁移、加密或更强的文件恢复保证时，使用 `VersionedJSONCodec`、`AESGCMCodec`、`Journal`、`Durable` 和 `WithConflictDetection`。详见 [`docs/architecture.md`](docs/architecture.md)。

## 完整示例

[`examples/workbench`](examples/workbench/README.md) 是完整的 Nodra + MyGo 桌面项目，包含克隆状态、本地草稿、带校验和备份的持久化、可防抖和取消的异步搜索、异步保存、Scope 关闭、原生刷新、英简双语和可选 DevTools。

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
```

较小的示例每个只演示一种集成：

- [`examples/basic`](examples/basic/main.go)：Store、订阅和批量更新。
- [`examples/mygo`](examples/mygo/main.go)：原生窗口、刷新，以及 Edit 菜单的撤销和重做。
- [`examples/persistent`](examples/persistent/main.go)：迁移、备份和异步写入持久化。
- [`examples/devtools`](examples/devtools/main.go)：本地检查器和快照恢复。
- [`examples/web`](examples/web/main.go)：原生窗口和网页通过桥接共享同一个 Store。

## DevTools

```go
inspector := devtools.New(devtools.Options{})
defer inspector.Close()
stop, err := devtools.Attach(inspector, "state", store.ReadOnly())
if err != nil {
	panic(err)
}
defer stop()
url, _ := inspector.Listen()
fmt.Println(url)
```

DevTools 默认只读，设置 `AllowRestore` 才会启用类型安全的 JSON 编辑和快照恢复。脱敏、限制、远程只读模式和 JSON 兼容性约定详见 [`docs/devtools.md`](docs/devtools.md)。

## 文档

- [`docs/public-api.md`](docs/public-api.md)：根包 API 索引。
- [`docs/bridge.md`](docs/bridge.md)：跟随 Store 并派发 Action 的网页。
- [`docs/architecture.md`](docs/architecture.md)：所有权、并发和生命周期语义。
- [`docs/compatibility.md`](docs/compatibility.md)：v0.x 到 v1 兼容策略。
- [`docs/support.md`](docs/support.md)：Go、MyGo、操作系统、架构和文件系统支持矩阵。
- [`CONTRIBUTING.md`](CONTRIBUTING.md)：开发和验证流程。
- [`SECURITY.md`](SECURITY.md)：漏洞报告和敏感数据政策。

## 验证

```sh
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Nodra 是一个 Go 模块：`github.com/yldm-tech/nodra`。项目要求 Go `1.27.3` 或更新版本，原生集成使用 MyGo `v0.7.4`。

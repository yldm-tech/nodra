# Nodra

[![Website](https://img.shields.io/badge/website-nodra.yldm.ai-c8f05a)](https://nodra.yldm.ai)
[![Go Reference](https://pkg.go.dev/badge/github.com/yldm-tech/nodra.svg)](https://pkg.go.dev/github.com/yldm-tech/nodra)
[![Latest version](https://img.shields.io/github/v/release/yldm-tech/nodra)](https://github.com/yldm-tech/nodra/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/yldm-tech/nodra)](go.mod)
[![GitHub stars](https://img.shields.io/github/stars/yldm-tech/nodra?style=flat)](https://github.com/yldm-tech/nodra/stargazers)
[![License](https://img.shields.io/github/license/yldm-tech/nodra)](LICENSE)
[![X](https://img.shields.io/badge/X-%40suzukaze__haduki-000000?logo=x&logoColor=white)](https://x.com/suzukaze_haduki)
[![GitHub Sponsors](https://img.shields.io/badge/GitHub%20Sponsors-ea4aaa?logo=githubsponsors&logoColor=white)](https://github.com/sponsors/houko)

Nodra は Go デスクトップアプリ向けの並行安全なリアクティブ状態ライブラリです。[MyGo](https://github.com/egoist/mygo) で Zustand に似た操作感を提供し、状態をレンダラーや永続化バックエンドから分離します。

[English](README.md) · [简体中文](README.zh-CN.md) · [한국어](README.ko.md) · [Español](README.es.md)

## インストール

```sh
go get github.com/yldm-tech/nodra@latest
```

コアパッケージは `github.com/yldm-tech/nodra` です。任意のローカルインスペクターは `github.com/yldm-tech/nodra/devtools`、任意の Web ページ用ブリッジは `github.com/yldm-tech/nodra/bridge`、ネイティブ UI 用の Edit メニュー連携は `github.com/yldm-tech/nodra/nativeui` です。

## クイックスタート

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

## 主な機能

- バージョン付き `Store[T]`、同期購読、上限付き Watch、読み取り専用ビュー。
- 派生状態、セレクター、投影、フィールドバインディング、アトミックトランザクション。
- グループ化、フィルター、サイズ予算に対応した Undo/Redo 履歴。
- JSON、スキーマ移行、暗号化、バックアップ、ジャーナル復旧、競合検出付き永続化。
- デバウンス、キャンセル、キュー、進捗、リトライ、メトリクス付き非同期 Action。
- `Scope` による所有権と終了順序の管理。
- 任意の MyGo 再描画ヘルパーと認証付きローカル DevTools インスペクター。
- MyGo アプリの Web ページが Store を購読し、登録済み Action で変更できるブリッジ。
- ネイティブ UI ウィンドウの取り消し/やり直しのショートカットと Edit メニューを History に接続。

## MyGo

```go
stop := nodra.InvalidateSelector(store, window, func(s State) int {
	return s.Count
})
defer stop()
```

MyGo の対象バージョンは `v0.7.4` です。一時的なウィジェット状態はビュー所有者に置き、共有 Store には編集したフィールドだけを書き戻します。最小のネイティブウィンドウの例は [`examples/mygo`](examples/mygo/main.go) を参照してください。

```go
history := nodra.Track(store, 100)
nativeui.InvalidateHistory(history, window) // redraw when Undo or Redo becomes possible

func (app *App) view(c *ui.Context) {
	nativeui.UndoShortcuts(c, app.history) // Cmd+Z, Shift+Cmd+Z and the Edit menu
	// ...
}
```

オプションの `nativeui` サブパッケージは、ウィンドウの取り消しとやり直しを `History` に結び付けます。ショートカットと、MyGo 0.7.4 以降の macOS では Edit メニューの項目も対象で、履歴が操作できるときだけ有効になります。フォーカス中のテキスト入力は独自の取り消しを保ちます。コアから分かれているため、ネイティブ UI を使わないアプリは MyGo の `ui` パッケージをリンクしません。

Store を共有するだけでウィンドウを開かないサービスや CLI は `-tags nodra_nomygo` でビルドできます。`*mygo.Window` を受け取るヘルパーが除かれ（`InvalidateTarget` は残ります）、MyGo がリンクされないため、`CGO_ENABLED=0` の Linux ビルドは静的実行ファイルになります。このタグはビルド全体に効きます。`nativeui` や、これらのヘルパーを呼ぶライブラリはこのタグではコンパイルできず、`bridge` を使うと MyGo が再びリンクされます。詳しくは [`docs/support.md`](docs/support.md) を参照してください。

## Web ページ

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

オプションの `bridge` サブパッケージにより、1 つの Store をネイティブウィンドウと Web ページ共通の唯一の状態源にできます。ページは投影が返す値だけを受け取り、値が変わったときにだけ新しいフレームを受け取ります（処理が遅いページで未取得のフレームが約 1 MiB たまると、最新のフレームにまとめられます）。状態は Go が登録した Action でのみ変更できます。[`bridge/client`](bridge/client/nodra.js) の JavaScript クライアントは依存関係がありません。詳しくは [`docs/bridge.md`](docs/bridge.md) と [`examples/web`](examples/web/main.go) を参照してください。

## 永続化

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

アプリケーションにマイグレーション、暗号化、より強いファイル復旧の保証が必要な場合は、`VersionedJSONCodec`、`AESGCMCodec`、`Journal`、`Durable`、`WithConflictDetection` を使います。詳しくは [`docs/architecture.md`](docs/architecture.md) を参照してください。

## 完全なサンプル

[`examples/workbench`](examples/workbench/README.md) は、クローン状態、ローカル下書き、検証付きバックアップ永続化、デバウンス可能なキャンセル対応検索、非同期保存、Scope、ネイティブ再描画、英語/簡体字中国語、DevTools を含む完全な Nodra + MyGo アプリです。

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
```

小さなサンプルはそれぞれ 1 つの連携を扱います。

- [`examples/basic`](examples/basic/main.go)：Store、購読、バッチ。
- [`examples/mygo`](examples/mygo/main.go)：ネイティブウィンドウ、再描画、Edit メニューの取り消しとやり直し。
- [`examples/persistent`](examples/persistent/main.go)：マイグレーション、バックアップ、ライトビハインド永続化。
- [`examples/devtools`](examples/devtools/main.go)：ローカルインスペクターと復元。
- [`examples/web`](examples/web/main.go)：ブリッジを通じて 1 つの Store を共有するネイティブウィンドウと Web ページ。

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

DevTools はデフォルトで読み取り専用です。`AllowRestore` を明示すると型検証付き JSON 編集とスナップショット復元を有効にできます。秘匿化、上限、リモート読み取り専用モード、JSON 互換性の契約については [`docs/devtools.md`](docs/devtools.md) を参照してください。

## ドキュメント

- [`docs/public-api.md`](docs/public-api.md)：ルートパッケージの API 索引。
- [`docs/bridge.md`](docs/bridge.md)：Store に追従し Action をディスパッチする Web ページ。
- [`docs/architecture.md`](docs/architecture.md)：所有権、並行性、ライフサイクル。
- [`docs/compatibility.md`](docs/compatibility.md)：v0.x から v1 の互換性方針。
- [`docs/support.md`](docs/support.md)：Go、MyGo、OS、アーキテクチャ、ファイルシステムの対応表。
- [`CONTRIBUTING.md`](CONTRIBUTING.md)：開発と検証の手順。
- [`SECURITY.md`](SECURITY.md)：脆弱性の報告と機密データの方針。

## 検証

```sh
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Nodra は 1 つの Go モジュール `github.com/yldm-tech/nodra` です。Go `1.27.3` 以上が必要で、ネイティブ連携には MyGo `v0.7.4` を使用します。

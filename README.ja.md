# Nodra

[![Website](https://img.shields.io/badge/website-nodra.yldm.ai-c8f05a)](https://nodra.yldm.ai)
[![Go Reference](https://pkg.go.dev/badge/github.com/yldm-tech/nodra.svg)](https://pkg.go.dev/github.com/yldm-tech/nodra)
[![Latest version](https://img.shields.io/github/v/release/yldm-tech/nodra)](https://github.com/yldm-tech/nodra/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/yldm-tech/nodra)](go.mod)
[![GitHub stars](https://img.shields.io/github/stars/yldm-tech/nodra?style=flat)](https://github.com/yldm-tech/nodra/stargazers)
[![License](https://img.shields.io/github/license/yldm-tech/nodra)](LICENSE)

Nodra は Go デスクトップアプリ向けの並行安全なリアクティブ状態ライブラリです。[MyGo](https://github.com/egoist/mygo) で Zustand に似た操作感を提供し、状態をレンダラーや永続化バックエンドから分離します。

[English](README.md) · [简体中文](README.zh-CN.md) · [한국어](README.ko.md) · [Español](README.es.md)

## インストール

```sh
go get github.com/yldm-tech/nodra@latest
```

コアパッケージは `github.com/yldm-tech/nodra`、任意のローカルインスペクターは `github.com/yldm-tech/nodra/devtools` です。

## クイックスタート

```go
store := nodra.New(State{})
defer store.Close()
stop := store.Subscribe(func(s nodra.Snapshot[State]) { fmt.Println(s.Value) })
defer stop()
_ = store.Update(func(s *State) { s.Count++ })
```

## 主な機能

- バージョン付き `Store[T]`、同期購読、上限付き Watch、読み取り専用ビュー。
- 派生状態、セレクター、投影、フィールドバインディング、アトミックトランザクション。
- グループ化、フィルター、サイズ予算に対応した Undo/Redo 履歴。
- JSON、スキーマ移行、暗号化、バックアップ、ジャーナル復旧、競合検出付き永続化。
- デバウンス、キャンセル、キュー、進捗、リトライ、メトリクス付き非同期 Action。
- `Scope` による所有権と終了順序の管理。
- MyGo の再描画バインディングと認証付きローカル DevTools。

## MyGo

```go
stop := nodra.InvalidateSelector(store, window, func(s State) int { return s.Count })
defer stop()
```

MyGo の対象バージョンは `v0.6.3` です。一時的なウィジェット状態はビュー所有者に置き、共有状態には編集したフィールドだけを書き戻します。最小例は [`examples/mygo`](examples/mygo/main.go) を参照してください。

## 完全なサンプル

[`examples/workbench`](examples/workbench/README.md) は、クローン状態、ローカル下書き、検証付きバックアップ永続化、デバウンス可能なキャンセル対応検索、非同期保存、Scope、ネイティブ再描画、英語/簡体字中国語、DevTools を含む完全な Nodra + MyGo アプリです。

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
```

## DevTools とドキュメント

DevTools はデフォルトで読み取り専用です。`AllowRestore` を明示すると型検証付き JSON 編集とスナップショット復元を有効にできます。詳しくは [`docs/devtools.md`](docs/devtools.md) を参照してください。

- [`docs/public-api.md`](docs/public-api.md)：公開 API 索引。
- [`docs/architecture.md`](docs/architecture.md)：所有権、並行性、ライフサイクル。
- [`docs/compatibility.md`](docs/compatibility.md)：v0.x から v1 の互換性方針。
- [`docs/support.md`](docs/support.md)：Go、MyGo、OS、アーキテクチャの対応表。

## 検証

```sh
go test ./...
go test -race ./...
go vet ./...
```

Go `1.27.3` 以上と MyGo `v0.6.3` を使用します。

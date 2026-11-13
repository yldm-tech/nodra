# Nodra

[![Website](https://img.shields.io/badge/website-nodra.yldm.ai-c8f05a)](https://nodra.yldm.ai)
[![Go Reference](https://pkg.go.dev/badge/github.com/yldm-tech/nodra.svg)](https://pkg.go.dev/github.com/yldm-tech/nodra)
[![Latest version](https://img.shields.io/github/v/release/yldm-tech/nodra)](https://github.com/yldm-tech/nodra/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/yldm-tech/nodra)](go.mod)
[![GitHub stars](https://img.shields.io/github/stars/yldm-tech/nodra?style=flat)](https://github.com/yldm-tech/nodra/stargazers)
[![License](https://img.shields.io/github/license/yldm-tech/nodra)](LICENSE)
[![X](https://img.shields.io/badge/X-%40suzukaze__haduki-000000?logo=x&logoColor=white)](https://x.com/suzukaze_haduki)
[![GitHub Sponsors](https://img.shields.io/badge/GitHub%20Sponsors-ea4aaa?logo=githubsponsors&logoColor=white)](https://github.com/sponsors/houko)

Nodra는 Go 데스크톱 애플리케이션을 위한 동시성 안전 반응형 상태 라이브러리입니다. [MyGo](https://github.com/egoist/mygo)에서 Zustand와 비슷한 사용성을 제공하며 상태를 렌더러와 영속성 백엔드에서 분리합니다.

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md) · [Español](README.es.md)

## 설치

```sh
go get github.com/yldm-tech/nodra@latest
```

핵심 패키지는 `github.com/yldm-tech/nodra`입니다. 선택적 로컬 인스펙터는 `github.com/yldm-tech/nodra/devtools`, 웹 페이지용 선택적 브리지는 `github.com/yldm-tech/nodra/bridge`, 네이티브 UI용 Edit 메뉴 통합은 `github.com/yldm-tech/nodra/nativeui`입니다.

## 빠른 시작

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

## 주요 기능

- 버전이 있는 `Store[T]`, 동기 구독, 제한된 Watch, 읽기 전용 뷰.
- 파생 상태, 선택기, 프로젝션, 필드 바인딩, 원자적 트랜잭션.
- 그룹화, 필터링, 크기 예산을 지원하는 Undo/Redo 기록.
- JSON, 스키마 마이그레이션, 암호화, 백업, 저널 복구, 충돌 감지 영속성.
- 디바운스, 취소, 큐, 진행률, 재시도, 메트릭을 지원하는 비동기 Action.
- `Scope`를 통한 소유권과 종료 순서 관리.
- 선택적 MyGo 무효화 헬퍼와 인증된 로컬 DevTools 인스펙터.
- MyGo 앱의 웹 페이지가 Store를 구독하고 등록된 Action으로 변경할 수 있는 브리지.
- 네이티브 UI 창의 실행 취소/다시 실행 단축키와 Edit 메뉴를 History에 연결.

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

MyGo 대상 버전은 `v0.7.4`입니다. 임시 위젯 상태는 뷰 소유자에 두고 공유 Store에는 수정된 필드만 반영하세요. 최소 네이티브 창 예제는 [`examples/mygo`](examples/mygo/main.go)에서 확인할 수 있습니다.

```go
history := nodra.Track(store, 100)
nativeui.InvalidateHistory(history, window) // redraw when Undo or Redo becomes possible

func (app *App) view(c *ui.Context) {
	nativeui.UndoShortcuts(c, app.history) // Cmd+Z, Shift+Cmd+Z and the Edit menu
	// ...
}
```

선택적 `nativeui` 하위 패키지는 창의 실행 취소와 다시 실행을 `History`에 연결합니다. 단축키와, MyGo 0.7.4부터 macOS에서는 Edit 메뉴 항목도 포함되며, 기록이 실행할 수 있을 때만 활성화됩니다. 포커스된 텍스트 입력은 자체 실행 취소를 유지합니다. 코어와 분리되어 있어 네이티브 UI가 없는 앱은 MyGo의 `ui` 패키지를 링크하지 않습니다.

Store를 공유하지만 창을 열지 않는 서비스와 CLI는 `-tags nodra_nomygo`로 빌드할 수 있습니다. `*mygo.Window`를 받는 헬퍼가 빠지고(`InvalidateTarget`은 남습니다) MyGo가 링크되지 않으므로, `CGO_ENABLED=0` Linux 빌드는 정적 실행 파일이 됩니다. 이 태그는 빌드 전체에 적용됩니다. `nativeui`나 이 헬퍼를 호출하는 라이브러리는 이 태그로 컴파일되지 않으며, `bridge`는 MyGo를 다시 링크합니다. 자세한 내용은 [`docs/support.md`](docs/support.md)를 참고하세요.

## 웹 페이지

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

선택적 `bridge` 하위 패키지는 하나의 Store를 네이티브 창과 웹 페이지 모두의 단일 상태 원천으로 만듭니다. 페이지는 프로젝션이 반환하는 값만 보고, 값이 바뀔 때만 새 프레임을 받으며(느린 페이지에 가져가지 않은 프레임이 약 1 MiB 쌓이면 최신 프레임으로 합쳐집니다), Go가 등록한 Action으로만 상태를 바꿀 수 있습니다. [`bridge/client`](bridge/client/nodra.js)의 JavaScript 클라이언트는 의존성이 없습니다. 자세한 내용은 [`docs/bridge.md`](docs/bridge.md)와 [`examples/web`](examples/web/main.go)를 참고하세요.

## 영속성

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

애플리케이션에 마이그레이션, 암호화 또는 더 강한 파일 복구 보장이 필요하면 `VersionedJSONCodec`, `AESGCMCodec`, `Journal`, `Durable`, `WithConflictDetection`을 사용하세요. 자세한 내용은 [`docs/architecture.md`](docs/architecture.md)를 참고하세요.

## 완전한 예제

[`examples/workbench`](examples/workbench/README.md)는 복제 상태, 로컬 초안, 검증된 백업 영속성, 디바운스 및 취소 가능한 검색, 비동기 저장, Scope, 네이티브 무효화, 영어/중국어 간체, DevTools를 포함한 완전한 Nodra + MyGo 데스크톱 프로젝트입니다.

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
```

더 작은 예제는 각각 하나의 통합을 다룹니다.

- [`examples/basic`](examples/basic/main.go): Store, 구독, 배치.
- [`examples/mygo`](examples/mygo/main.go): 네이티브 창, 무효화, Edit 메뉴의 실행 취소와 다시 실행.
- [`examples/persistent`](examples/persistent/main.go): 마이그레이션, 백업, 지연 쓰기(write-behind) 영속성.
- [`examples/devtools`](examples/devtools/main.go): 로컬 인스펙터와 복원.
- [`examples/web`](examples/web/main.go): 브리지를 통해 하나의 Store를 공유하는 네이티브 창과 웹 페이지.

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

DevTools는 기본적으로 읽기 전용입니다. `AllowRestore`를 명시하면 타입 검증 JSON 편집과 스냅샷 복원을 사용할 수 있습니다. 민감 정보 가림, 제한, 원격 읽기 전용 모드, JSON 호환성 계약은 [`docs/devtools.md`](docs/devtools.md)를 참고하세요.

## 문서

- [`docs/public-api.md`](docs/public-api.md): 루트 패키지 API 색인.
- [`docs/bridge.md`](docs/bridge.md): Store를 따르고 Action을 디스패치하는 웹 페이지.
- [`docs/architecture.md`](docs/architecture.md): 소유권, 동시성, 생명주기 의미.
- [`docs/compatibility.md`](docs/compatibility.md): v0.x에서 v1로의 호환성 정책.
- [`docs/support.md`](docs/support.md): Go, MyGo, OS, 아키텍처, 파일 시스템 지원 표.
- [`CONTRIBUTING.md`](CONTRIBUTING.md): 개발 및 검증 절차.
- [`SECURITY.md`](SECURITY.md): 취약점 보고와 민감 데이터 정책.

## 검증

```sh
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Nodra는 하나의 Go 모듈 `github.com/yldm-tech/nodra`입니다. Go `1.27.3` 이상이 필요하며 네이티브 통합에는 MyGo `v0.7.4`를 사용합니다.

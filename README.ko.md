# Nodra

[![Website](https://img.shields.io/badge/website-nodra.yldm.ai-c8f05a)](https://nodra.yldm.ai)
[![Go Reference](https://pkg.go.dev/badge/github.com/yldm-tech/nodra.svg)](https://pkg.go.dev/github.com/yldm-tech/nodra)
[![Latest version](https://img.shields.io/github/v/release/yldm-tech/nodra)](https://github.com/yldm-tech/nodra/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/yldm-tech/nodra)](go.mod)
[![GitHub stars](https://img.shields.io/github/stars/yldm-tech/nodra?style=flat)](https://github.com/yldm-tech/nodra/stargazers)
[![License](https://img.shields.io/github/license/yldm-tech/nodra)](LICENSE)

Nodra는 Go 데스크톱 애플리케이션을 위한 동시성 안전 반응형 상태 라이브러리입니다. [MyGo](https://github.com/egoist/mygo)에서 Zustand와 비슷한 사용성을 제공하며 상태를 렌더러와 영속성 백엔드에서 분리합니다.

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md) · [Español](README.es.md)

## 설치

```sh
go get github.com/yldm-tech/nodra@latest
```

핵심 패키지는 `github.com/yldm-tech/nodra`이며 선택적 로컬 인스펙터는 `github.com/yldm-tech/nodra/devtools`입니다.

## 빠른 시작

```go
store := nodra.New(State{})
defer store.Close()
stop := store.Subscribe(func(s nodra.Snapshot[State]) { fmt.Println(s.Value) })
defer stop()
_ = store.Update(func(s *State) { s.Count++ })
```

## 주요 기능

- 버전이 있는 `Store[T]`, 동기 구독, 제한된 Watch, 읽기 전용 뷰.
- 파생 상태, 선택기, 프로젝션, 필드 바인딩, 원자적 트랜잭션.
- 그룹화, 필터링, 크기 예산을 지원하는 Undo/Redo 기록.
- JSON, 스키마 마이그레이션, 암호화, 백업, 저널 복구, 충돌 감지 영속성.
- 디바운스, 취소, 큐, 진행률, 재시도, 메트릭을 지원하는 비동기 Action.
- `Scope`를 통한 소유권과 종료 순서 관리.
- MyGo 무효화 헬퍼와 인증된 로컬 DevTools.

## MyGo

```go
stop := nodra.InvalidateSelector(store, window, func(s State) int { return s.Count })
defer stop()
```

MyGo 대상 버전은 `v0.6.3`입니다. 임시 위젯 상태는 뷰 소유자에 두고 공유 Store에는 수정된 필드만 반영하세요. 최소 예제는 [`examples/mygo`](examples/mygo/main.go)에서 확인할 수 있습니다.

## 완전한 예제

[`examples/workbench`](examples/workbench/README.md)는 복제 상태, 로컬 초안, 검증된 백업 영속성, 디바운스 및 취소 가능한 검색, 비동기 저장, Scope, 네이티브 무효화, 영어/중국어 간체, DevTools를 포함한 완전한 Nodra + MyGo 데스크톱 프로젝트입니다.

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
```

## DevTools 및 문서

DevTools는 기본적으로 읽기 전용입니다. `AllowRestore`를 명시하면 타입 검증 JSON 편집과 스냅샷 복원을 사용할 수 있습니다. 자세한 내용은 [`docs/devtools.md`](docs/devtools.md)를 참고하세요.

- [`docs/public-api.md`](docs/public-api.md): 공개 API 색인.
- [`docs/architecture.md`](docs/architecture.md): 소유권, 동시성, 생명주기 의미.
- [`docs/compatibility.md`](docs/compatibility.md): v0.x에서 v1로의 호환성 정책.
- [`docs/support.md`](docs/support.md): Go, MyGo, OS, 아키텍처 지원 표.

## 검증

```sh
go test ./...
go test -race ./...
go vet ./...
```

Go `1.27.3` 이상과 MyGo `v0.6.3`을 사용합니다.

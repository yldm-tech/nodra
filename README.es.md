# Nodra

[![Website](https://img.shields.io/badge/website-nodra.yldm.ai-c8f05a)](https://nodra.yldm.ai)
[![Go Reference](https://pkg.go.dev/badge/github.com/yldm-tech/nodra.svg)](https://pkg.go.dev/github.com/yldm-tech/nodra)
[![Latest version](https://img.shields.io/github/v/release/yldm-tech/nodra)](https://github.com/yldm-tech/nodra/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/yldm-tech/nodra)](go.mod)
[![GitHub stars](https://img.shields.io/github/stars/yldm-tech/nodra?style=flat)](https://github.com/yldm-tech/nodra/stargazers)
[![License](https://img.shields.io/github/license/yldm-tech/nodra)](LICENSE)

Nodra es una biblioteca de estado reactivo y segura frente a concurrencia para aplicaciones de escritorio Go. Ofrece una experiencia parecida a Zustand para [MyGo](https://github.com/egoist/mygo) y mantiene el estado separado del renderizador y del backend de persistencia.

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

## Instalación

```sh
go get github.com/yldm-tech/nodra@latest
```

El paquete principal es `github.com/yldm-tech/nodra` y el inspector local opcional es `github.com/yldm-tech/nodra/devtools`.

## Inicio rápido

```go
store := nodra.New(State{})
defer store.Close()
stop := store.Subscribe(func(s nodra.Snapshot[State]) { fmt.Println(s.Value) })
defer stop()
_ = store.Update(func(s *State) { s.Count++ })
```

## Capacidades principales

- `Store[T]` versionado con suscripciones síncronas, Watch limitado y vistas de solo lectura.
- Estado derivado, selectores, proyecciones, enlaces de campos y transacciones atómicas.
- Historial Undo/Redo con grupos, filtros y presupuesto de tamaño.
- Persistencia JSON, migraciones, cifrado, copias de seguridad, diario y detección de conflictos.
- Acciones asíncronas con debounce, cancelación, cola, progreso, reintentos y métricas.
- Gestión de propiedad y orden de cierre mediante `Scope`.
- Integración de invalidación con MyGo e inspector DevTools local autenticado.

## MyGo

```go
stop := nodra.InvalidateSelector(store, window, func(s State) int { return s.Count })
defer stop()
```

La integración usa MyGo `v0.7.2`. Mantén el estado temporal de los widgets en el propietario de la vista y escribe en el Store compartido solo el campo editado. Consulta [`examples/mygo`](examples/mygo/main.go) para un ejemplo mínimo.

## Ejemplo completo

[`examples/workbench`](examples/workbench/README.md) es el proyecto completo Nodra + MyGo: estado clonado, borradores locales, persistencia validada con copias, búsqueda asíncrona cancelable con debounce, guardado asíncrono, Scope, invalidación nativa, inglés/chino simplificado y DevTools opcional.

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
```

## DevTools y documentación

DevTools es de solo lectura por defecto. `AllowRestore` activa explícitamente la edición JSON tipada y la restauración de instantáneas. Consulta [`docs/devtools.md`](docs/devtools.md) para más detalles.

- [`docs/public-api.md`](docs/public-api.md): índice de la API pública.
- [`docs/architecture.md`](docs/architecture.md): propiedad, concurrencia y ciclo de vida.
- [`docs/compatibility.md`](docs/compatibility.md): política de compatibilidad de v0.x a v1.
- [`docs/support.md`](docs/support.md): matriz de Go, MyGo, sistemas y arquitecturas compatibles.

## Verificación

```sh
go test ./...
go test -race ./...
go vet ./...
```

El proyecto requiere Go `1.27.3+` y usa MyGo `v0.7.2` para la integración nativa.

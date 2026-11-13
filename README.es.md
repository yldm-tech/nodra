# Nodra

[![Website](https://img.shields.io/badge/website-nodra.yldm.ai-c8f05a)](https://nodra.yldm.ai)
[![Go Reference](https://pkg.go.dev/badge/github.com/yldm-tech/nodra.svg)](https://pkg.go.dev/github.com/yldm-tech/nodra)
[![Latest version](https://img.shields.io/github/v/release/yldm-tech/nodra)](https://github.com/yldm-tech/nodra/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/yldm-tech/nodra)](go.mod)
[![GitHub stars](https://img.shields.io/github/stars/yldm-tech/nodra?style=flat)](https://github.com/yldm-tech/nodra/stargazers)
[![License](https://img.shields.io/github/license/yldm-tech/nodra)](LICENSE)
[![X](https://img.shields.io/badge/X-%40suzukaze__haduki-000000?logo=x&logoColor=white)](https://x.com/suzukaze_haduki)
[![GitHub Sponsors](https://img.shields.io/badge/GitHub%20Sponsors-ea4aaa?logo=githubsponsors&logoColor=white)](https://github.com/sponsors/houko)

Nodra es una biblioteca de estado reactivo y segura frente a concurrencia para aplicaciones de escritorio Go. Ofrece una experiencia parecida a Zustand para [MyGo](https://github.com/egoist/mygo) y mantiene el estado separado del renderizador y del backend de persistencia.

[English](README.md) · [简体中文](README.zh-CN.md) · [日本語](README.ja.md) · [한국어](README.ko.md)

## Instalación

```sh
go get github.com/yldm-tech/nodra@latest
```

El paquete principal es `github.com/yldm-tech/nodra`. El inspector opcional es `github.com/yldm-tech/nodra/devtools`, el puente opcional hacia páginas web es `github.com/yldm-tech/nodra/bridge` y la integración con el menú Edit para la UI nativa es `github.com/yldm-tech/nodra/nativeui`.

## Inicio rápido

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

## Capacidades principales

- `Store[T]` versionado con suscripciones síncronas, Watch limitado y vistas de solo lectura.
- Estado derivado, selectores, proyecciones, enlaces de campos y transacciones atómicas.
- Historial Undo/Redo con grupos, filtros y presupuesto de tamaño.
- Persistencia JSON, migraciones, cifrado, copias de seguridad, diario y detección de conflictos.
- Acciones asíncronas con debounce, cancelación, cola, progreso, reintentos y métricas.
- Gestión de propiedad y orden de cierre mediante `Scope`.
- Helpers opcionales de invalidación para MyGo e inspector DevTools local autenticado.
- Un puente para que las páginas web de una app MyGo sigan un Store y lo cambien mediante acciones registradas.
- Deshacer y Rehacer de los atajos y del menú Edit de la ventana sobre un History, para la UI nativa de MyGo.

## MyGo

```go
stop := nodra.InvalidateSelector(store, window, func(s State) int {
	return s.Count
})
defer stop()
```

La integración usa MyGo `v0.7.4`. Mantén el estado temporal de los widgets en el propietario de la vista y escribe en el Store compartido solo el campo editado. Consulta [`examples/mygo`](examples/mygo/main.go) para una ventana nativa mínima.

```go
history := nodra.Track(store, 100)
nativeui.InvalidateHistory(history, window) // redraw when Undo or Redo becomes possible

func (app *App) view(c *ui.Context) {
	nativeui.UndoShortcuts(c, app.history) // Cmd+Z, Shift+Cmd+Z and the Edit menu
	// ...
}
```

El subpaquete opcional `nativeui` hace que Deshacer y Rehacer de la ventana actúen sobre un `History`: los atajos y, con MyGo 0.7.4 en macOS, los elementos del menú Edit, que solo se habilitan cuando el historial puede actuar. Un campo de texto con el foco conserva su propio deshacer. Está separado del núcleo para que las apps sin UI nativa no enlacen el paquete `ui` de MyGo.

Los servicios y CLI que comparten Stores pero nunca abren una ventana pueden compilarse con `-tags nodra_nomygo`. La etiqueta omite las funciones auxiliares que reciben un `*mygo.Window` (`InvalidateTarget` se mantiene), así que MyGo no se enlaza y una compilación de Linux con `CGO_ENABLED=0` produce un ejecutable estático. La etiqueta se aplica a toda la compilación: `nativeui`, o cualquier biblioteca que llame a esas funciones, no compila con ella, y `bridge` vuelve a enlazar MyGo. Consulta [`docs/support.md`](docs/support.md).

## Páginas web

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

El subpaquete opcional `bridge` convierte un Store en la única fuente de verdad para ventanas nativas y páginas web. Las páginas solo ven lo que devuelve la proyección, reciben un frame cuando cambia (agrupados en el más reciente solo cuando una página lenta acumula cerca de 1 MiB de frames sin recoger) y solo cambian el estado mediante las acciones que Go registra. El cliente JavaScript de [`bridge/client`](bridge/client/nodra.js) no tiene dependencias. Consulta [`docs/bridge.md`](docs/bridge.md) y [`examples/web`](examples/web/main.go).

## Persistencia

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

Usa `VersionedJSONCodec`, `AESGCMCodec`, `Journal`, `Durable` y `WithConflictDetection` cuando la aplicación necesite migraciones, cifrado o garantías más fuertes de recuperación de archivos. Los detalles están en [`docs/architecture.md`](docs/architecture.md).

## Ejemplo completo

[`examples/workbench`](examples/workbench/README.md) es el proyecto completo Nodra + MyGo: estado clonado, borradores locales, persistencia validada con copias, búsqueda asíncrona cancelable con debounce, guardado asíncrono, Scope, invalidación nativa, inglés/chino simplificado y DevTools opcional.

```sh
go run ./examples/workbench
go run ./examples/workbench -devtools
```

Los ejemplos más pequeños cubren una integración cada uno:

- [`examples/basic`](examples/basic/main.go): Store, suscripciones y lotes.
- [`examples/mygo`](examples/mygo/main.go): ventanas nativas, invalidación, y Deshacer y Rehacer del menú Edit.
- [`examples/persistent`](examples/persistent/main.go): migraciones, copias de seguridad y persistencia write-behind.
- [`examples/devtools`](examples/devtools/main.go): inspector local y restauración.
- [`examples/web`](examples/web/main.go): una ventana nativa y una página web que comparten un Store mediante el puente.

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

DevTools es de solo lectura por defecto. `AllowRestore` activa explícitamente la edición JSON tipada y la restauración de instantáneas. Consulta [`docs/devtools.md`](docs/devtools.md) para la ocultación de datos sensibles, los límites, el modo remoto de solo lectura y el contrato de compatibilidad JSON.

## Documentación

- [`docs/public-api.md`](docs/public-api.md): índice de la API del paquete raíz.
- [`docs/bridge.md`](docs/bridge.md): páginas web que siguen un Store y despachan acciones.
- [`docs/architecture.md`](docs/architecture.md): semántica de propiedad, concurrencia y ciclo de vida.
- [`docs/compatibility.md`](docs/compatibility.md): política de compatibilidad de v0.x a v1.
- [`docs/support.md`](docs/support.md): matriz de Go, MyGo, sistemas operativos, arquitecturas y sistemas de archivos compatibles.
- [`CONTRIBUTING.md`](CONTRIBUTING.md): flujo de desarrollo y verificación.
- [`SECURITY.md`](SECURITY.md): notificación de vulnerabilidades y política de datos sensibles.

## Verificación

```sh
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Nodra es un único módulo Go: `github.com/yldm-tech/nodra`. El proyecto requiere Go `1.27.3` o posterior y usa MyGo `v0.7.4` para la integración nativa.

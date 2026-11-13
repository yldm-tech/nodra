# Web pages

The bridge is an optional Go subpackage, `github.com/yldm-tech/nodra/bridge`, that lets the web pages of a MyGo app follow a Store and change it, so one Store in Go is the source of truth for windows of native UI and web pages alike. MyGo's native UI reads Go state directly and needs none of this: use the bridge for windows that show a web page, and for web views inside native UI.

Without it, an app keeps a second copy of its state in the frontend, in a JavaScript store, and keeps the two in step by hand with bound methods and events. With it, Go decides what each page may see and what it may do:

- A page sees a state only as Go exposes it, as a whole or through a projection that leaves secrets and large internals in Go.
- A page receives a new frame only when the value it sees changes. By default frames coalesce into the latest only once MyGo's channel holds back sends, at about 1 MiB of frames the page has not taken per watch, so a slow page may still receive each intermediate change. `Options.FrameInterval` sets the least time between two frames of a watch: changes within it coalesce into the latest, which is sent when it ends, and a change after a quiet spell goes at once. A source that closes still sends its final value, when the interval ends, before the watch ends with `ErrClosed`, so a page sees the last status of a task whose store closes right after it; stopping the exposure or closing the bridge ends a watch at once and drops a change it is holding back. For state that changes faster than a page needs to show it, such as progress, playback position or streamed results, set it to a display frame:

  ```go
  b := bridge.New(bridge.Options{FrameInterval: 16 * time.Millisecond})
  ```

- A page changes state only by dispatching an action Go registered; it never writes a store.

## Go

Expose sources and register actions, then bind the bridge before `App.Run`:

```go
// PageState is what pages see of State. encoding/json names an untagged field as it is declared, so the tag is what lets the page read state.count.
type PageState struct {
	Count int `json:"count"`
}

b := bridge.New(bridge.Options{})
defer b.Close()

// Pages see the count, not the token.
stop, err := bridge.ExposeProjection(b, "counter", store, func(s State) PageState {
	return PageState{Count: s.Count}
})
if err != nil {
	return err
}
defer stop()

stopAdd, err := bridge.Handle(b, "counter/add", func(ctx context.Context, arg struct{ By int }) error {
	return store.Update(func(s *State) { s.Count += arg.By })
})
if err != nil {
	return err
}
defer stopAdd()

b.Bind() // the service Nodra; BindAs("Other") for another name
```

`Expose` and `ExposeProjection` take any `Observable`: a `Store`, its `ReadOnly` view or a `Derived` value. The value pages receive is the source's value, or the projection's, encoded with `encoding/json`. Projections run on the goroutine of each page's watch, possibly for several pages at once, so they must be pure and concurrency-safe and treat the source's reference values as read-only.

`Handle` decodes the JSON argument a page dispatches into the handler's argument type, a missing or `null` one into its zero value, and calls the handler with the call's context, which MyGo cancels when the page navigates away or its window closes. The handler changes state through the app's stores like any other code, so the bridge adds no second way to write state, and history, persistence and DevTools see the change as usual. The handler's error rejects the page's promise.

State names and action names are separate: the same name may be both. A name is taken until its stop function runs or the bridge closes; `ErrExists` reports a second use. Until it closes, a bridge remembers every name a state was exposed under, so that a watch of a name exposed again sends its first frame (see the service below); an app that exposes a growing number of distinct names, such as one per document, keeps a small entry for each.

### Which pages may call

MyGo lets only the app's own pages call bound services: its frontend, custom protocols, `file:` and `about:` pages, and origins a window lists in `PageOptions.TrustedOrigins`. Frames embedded in a page never call. `Options.AllowRead` and `Options.AllowDispatch` narrow this per state and per action; their context is the call's, so `mygo.CallerWindow(ctx)` and `mygo.CallerPage(ctx)` tell which window and page are asking:

```go
b := bridge.New(bridge.Options{
	AllowDispatch: func(ctx context.Context, action string) bool {
		return mygo.CallerWindow(ctx) == editor
	},
})
```

A refused call fails with `ErrDenied`, whether or not anything is exposed or registered under the name, so a refused page cannot tell which names exist; the functions are therefore also asked about names nothing is exposed or registered under.

## JavaScript

The client is `bridge/client/nodra.js`, with its types in `nodra.d.ts`. It calls the bound service through `window.mygo`, which MyGo injects into every page, so it has no dependencies and runs with or without a bundler. Copy the two files into a frontend that has a build step; an app that serves its pages from Go can serve `bridge.Client` instead:

```go
mux := http.NewServeMux()
mux.Handle("/nodra/", http.StripPrefix("/nodra/", mygo.FileServer(bridge.Client)))
mux.Handle("/", mygo.FileServer(pages))
mygo.Protocol.Handle("mygo", mux)
```

```js
import { dispatch, remoteState } from "/nodra/nodra.js";

const counter = remoteState("counter");
const stop = counter.subscribe(() => {
  if (counter.hasValue()) render(counter.getSnapshot());
});

await dispatch("counter/add", { By: 1 });
```

`remoteState` asks nothing of Go until its first listener subscribes, and stops watching when the last one unsubscribes; the next subscriber resumes from the version the page holds. `getSnapshot` returns the same object until a new frame arrives, so the state fits React's `useSyncExternalStore`:

```tsx
const counter = remoteState<PageState>("counter");

function Count() {
  const state = useSyncExternalStore(counter.subscribe, counter.getSnapshot);
  return <output>{state?.count ?? "…"}</output>;
}
```

`ready()` resolves with the current value: the one a running watch keeps once it has sent a frame, or else the next frame of a watch Go sends its current frame to. It starts that watch when none runs, and restarts a watch that resumed from the version the page holds and has sent nothing yet, since Go sends such a watch no frame while the value is unchanged; subscribers follow the restarted watch. A value the page held from an ended watch is therefore not mistaken for the current one. `getError()` tells why a watch ended, such as a state that is not exposed (`ErrNotFound`), a page that is not allowed (`ErrDenied`) or a state that went away (`ErrClosed`); listeners are called then too, and a later subscriber tries again. `getState(name)` reads a frame once without following the state. Every function takes `{ service }` for a bridge bound with `BindAs`.

## The service

Apps that use `mygo generate` get the bridge's service in their typed client too. Its methods are:

| Method | What it does |
| --- | --- |
| `Get(name)` | returns the current `Frame` of a state |
| `Watch(name, after, frames)` | streams frames through a `Channel<Frame>`: the current one unless its version is `after` and the name was exposed only once, then one whenever the encoded value changes, until the page closes the channel or the state goes away |
| `Dispatch(name, arg)` | runs an action with its JSON argument |

A `Frame` is `{ name, version, value }`. Versions are the source's and only grow within a watch. A name exposed again may hold another source, whose versions count apart, so a watch of it sends its first frame even when the version equals `after`. The Go methods `Bridge.Get`, `Bridge.Watch` and `Bridge.Dispatch` do the same without MyGo, for tests and for Go code.

## Compatibility surface

The exported `bridge` Go API, the service's method names and arguments, and the `Frame` JSON fields are part of the v1 contract; fields may be added. The JavaScript client follows the same contract for its exported functions and the members of `RemoteState`.

[`examples/web`](../examples/web/main.go) shows a native window and a web page sharing one Store.

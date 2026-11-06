// Example web shows one Store behind a window of native UI and a window of a web page: both read the same count, and either changes it. The page follows a projection of the state through the bridge and changes it only through the actions Go registers.
package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yldm-tech/nodra"
	"github.com/yldm-tech/nodra/bridge"
)

//go:embed page
var pageFiles embed.FS

type appState struct {
	Count int
	// Token stays in Go: the projection the page sees leaves it out.
	Token string
}

// pageState is what the page sees of appState.
type pageState struct {
	Count int `json:"count"`
}

func project(s appState) pageState { return pageState{Count: s.Count} }

type step struct {
	By int `json:"by"`
}

func main() {
	store := nodra.New(appState{Token: "kept in Go"})
	defer func() { _ = store.Close() }()

	b, err := newBridge(store)
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()
	b.Bind()
	if err := mygo.Protocol.Handle("mygo", frontend()); err != nil {
		log.Fatal(err)
	}

	mygo.App.WhenReady(func() {
		native := mygo.NewWindow(mygo.WindowOptions{Title: "Native UI", Width: 360, Height: 260, Content: ui.View(func(c *ui.Context) { view(c, store) })})
		nodra.InvalidateSelector(store, native, func(s appState) int { return s.Count })
		mygo.NewWindow(mygo.WindowOptions{Title: "Web page", URL: "/", Width: 360, Height: 260})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// newBridge exposes the counter to pages and registers its actions.
func newBridge(store *nodra.Store[appState]) (*bridge.Bridge, error) {
	b := bridge.New(bridge.Options{})
	_, err1 := bridge.ExposeProjection(b, "counter", store, project)
	_, err2 := bridge.Handle(b, "counter/add", func(_ context.Context, arg step) error {
		return store.Update(func(s *appState) { s.Count += arg.By })
	})
	_, err3 := bridge.Handle(b, "counter/reset", func(context.Context, struct{}) error {
		return store.Update(func(s *appState) { s.Count = 0 })
	})
	if err := errors.Join(err1, err2, err3); err != nil {
		b.Close()
		return nil, err
	}
	return b, nil
}

// frontend serves the page, and the bridge's client under /nodra/.
func frontend() http.Handler {
	page, err := fs.Sub(pageFiles, "page")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/nodra/", http.StripPrefix("/nodra/", mygo.FileServer(bridge.Client)))
	mux.Handle("/", mygo.FileServer(page))
	return mux
}

func view(c *ui.Context, store *nodra.Store[appState]) {
	count := store.Get().Count
	ui.Column(c).Fill().Center().Padding(20).Gap(16).Children(func() {
		ui.Text(c, "Native UI").FontSize(18).Bold()
		ui.Textf(c, "Count: %d", count).FontSize(32)
		ui.Row(c).Gap(8).Children(func() {
			ui.Button(c, "Reset").OnClick(func() { _ = store.Update(func(s *appState) { s.Count = 0 }) })
			ui.PrimaryButton(c, "+1").OnClick(func() { _ = store.Update(func(s *appState) { s.Count++ }) })
		})
	})
}

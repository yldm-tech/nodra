// Package bridge exposes Nodra state to the web pages of a MyGo app, so a Store in Go stays the one source of truth for windows of native UI and web pages alike. It is separate from the core, as devtools is, so apps of native UI alone need not import it.
//
// The app exposes a source under a name, optionally projected to the value pages may see, registers the actions pages may dispatch, and binds the bridge before App.Run:
//
//	b := bridge.New(bridge.Options{})
//	stop, err := bridge.ExposeProjection(b, "counter", store, func(s State) PageState { return PageState{Count: s.Count} })
//	stopIncrement, err := bridge.Handle(b, "counter/increment", func(ctx context.Context, by int) error {
//		return store.Update(func(s *State) { s.Count += by })
//	})
//	b.Bind()
//
// Pages read and follow the state through the bound service, Nodra by default: Get returns the current frame, and Watch streams a frame through a MyGo channel each time the projected value changes, the latest one when the page is slower than the store. Dispatch runs a registered action with its JSON argument. Pages never write state directly. The JavaScript client in the repository's web directory wraps these calls.
package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/yldm-tech/nodra"
)

// DefaultService is the name Bind gives the bridge's service, which pages call as Nodra.Get, Nodra.Watch and Nodra.Dispatch.
const DefaultService = "Nodra"

var (
	// ErrClosed is returned once the bridge is closed, and ends a page's watch when its source is no longer exposed or has closed.
	ErrClosed = errors.New("nodra bridge: closed")
	// ErrNotFound is returned to a page the options allow for a state or an action name that nothing is exposed or registered under.
	ErrNotFound = errors.New("nodra bridge: state or action not found")
	// ErrExists is returned when a name is exposed or registered twice.
	ErrExists = errors.New("nodra bridge: name already in use")
	// ErrInvalid is returned for an empty name, a nil source or function, and an argument that does not decode.
	ErrInvalid = errors.New("nodra bridge: invalid name, source or argument")
	// ErrDenied is returned when Options.AllowRead or Options.AllowDispatch refuses the calling page, whether or not anything is exposed or registered under the name.
	ErrDenied = errors.New("nodra bridge: not allowed for this page")
)

// Options configures a Bridge. MyGo lets only the app's own pages call bound services (see TrustedOrigins in MyGo's bindings guide); AllowRead and AllowDispatch narrow that further, per state or action name. Their context is the call's, so mygo.CallerWindow and mygo.CallerPage tell which window and page call. Nil allows every page MyGo lets call. They are asked before the name is looked up, so they may be called with a name nothing is exposed or registered under, and a page they refuse gets ErrDenied either way. They must be concurrency-safe.
type Options struct {
	AllowRead     func(ctx context.Context, state string) bool
	AllowDispatch func(ctx context.Context, action string) bool
}

// Frame is a state as pages receive it: its name, the source version it was taken at and the projected value as JSON. Versions are the source's and only grow within a watch; a page that already holds a version passes it to Watch to skip a frame it has.
type Frame struct {
	Name    string          `json:"name"`
	Version uint64          `json:"version"`
	Value   json.RawMessage `json:"value"`
}

// Bridge holds the states and actions exposed to pages. Its methods are safe for concurrent use. Until it closes, it remembers every name a state was exposed under, so that a watch of a name exposed again sends its first frame (see Watch); an app that exposes a growing number of distinct names keeps a small entry for each.
type Bridge struct {
	options Options
	service *service

	mu      sync.Mutex
	closed  bool
	done    chan struct{}
	states  map[string]*exposure
	actions map[string]*action
	// exposed holds every name a state has been exposed under until the bridge closes, so a later exposure under the name knows that pages may hold versions of another source.
	exposed map[string]struct{}
}

// action is one registered action; its pointer tells a registration from a later one under the same name.
type action struct {
	run func(context.Context, json.RawMessage) error
}

// exposure is one exposed source, with its type erased.
type exposure struct {
	// read returns the source's current version and the projected value encoded as JSON.
	read func() (uint64, json.RawMessage, error)
	// subscribe calls changed after every snapshot the source publishes, and returns a function that stops it.
	subscribe func(changed func()) func()
	// sourceDone is closed when the source closes, or nil when the source cannot tell.
	sourceDone <-chan struct{}
	// stopped is closed when the exposure is stopped or the bridge closes.
	stopped chan struct{}
	// reused is set when the name was exposed before: versions count per source, so a version a page holds may be another source's, and Watch sends the first frame whatever after is.
	reused bool
}

// New returns an open Bridge with nothing exposed.
func New(options Options) *Bridge {
	b := &Bridge{
		options: options,
		done:    make(chan struct{}),
		states:  make(map[string]*exposure),
		actions: make(map[string]*action),
		exposed: make(map[string]struct{}),
	}
	b.service = &service{bridge: b}
	return b
}

// Bind binds the bridge's service under DefaultService. Like mygo.Bind, it must run before App.Run, and it panics when the name is taken.
func (b *Bridge) Bind() { b.BindAs(DefaultService) }

// BindAs binds the bridge's service under name, for an app that binds more than one bridge or already uses the name Nodra. Pages pass the same name to the client.
func (b *Bridge) BindAs(name string) { mygo.BindAs(name, b.service) }

// Close stops every exposure, which ends the pages' watches with ErrClosed, and makes later calls fail with ErrClosed. Sources stay open: the app owns them. Close is safe to call more than once.
func (b *Bridge) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	close(b.done)
	for name, e := range b.states {
		close(e.stopped)
		delete(b.states, name)
	}
	clear(b.actions)
	// Nothing is exposed after Close, so the names exposed before are not needed.
	clear(b.exposed)
}

// Expose makes source readable by pages under name, as it is: its value must encode as JSON, and every field the encoding includes reaches the pages. Use ExposeProjection to send them only part of it. The returned function stops the exposure, which ends the pages' watches of it; it is safe to call more than once.
func Expose[T any](b *Bridge, name string, source nodra.Observable[T]) (stop func(), err error) {
	return ExposeProjection(b, name, source, func(value T) T { return value })
}

// ExposeProjection makes the projection of source readable by pages under name. Pages see only what project returns, encoded as JSON, so secrets and large internals can stay in Go. A page receives a frame only when the encoded projection changes, so changes to fields the projection leaves out send nothing. project runs on the goroutine of each page's watch, possibly for several pages at once: it must be pure, concurrency-safe and treat the source's reference values as read-only.
func ExposeProjection[T, V any](b *Bridge, name string, source nodra.Observable[T], project func(T) V) (stop func(), err error) {
	if b == nil || !validName(name) || isNil(source) || project == nil {
		return nil, ErrInvalid
	}
	e := &exposure{
		read: func() (uint64, json.RawMessage, error) {
			snapshot := source.Snapshot()
			raw, err := json.Marshal(project(snapshot.Value))
			if err != nil {
				return snapshot.Version, nil, fmt.Errorf("nodra bridge: encode %q: %w", name, err)
			}
			return snapshot.Version, raw, nil
		},
		subscribe: func(changed func()) func() {
			return source.SubscribeSnapshot(func(nodra.Snapshot[T]) { changed() })
		},
		stopped: make(chan struct{}),
	}
	if closer, ok := source.(interface{ Done() <-chan struct{} }); ok {
		e.sourceDone = closer.Done()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, ErrClosed
	}
	if _, taken := b.states[name]; taken {
		return nil, ErrExists
	}
	_, e.reused = b.exposed[name]
	b.exposed[name] = struct{}{}
	b.states[name] = e
	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			if b.states[name] == e {
				delete(b.states, name)
				close(e.stopped)
			}
		})
	}, nil
}

// Handle registers an action pages may dispatch under name. A dispatch decodes its JSON argument into an A, a missing or null one into A's zero value, and calls handle on the dispatching call's goroutine, with the call's context, which is canceled when the page navigates away or its window closes. The error handle returns rejects the page's promise. handle changes state through the app's stores as any other code does; the pages that watch it receive the result. The returned function unregisters the action and is safe to call more than once.
func Handle[A any](b *Bridge, name string, handle func(ctx context.Context, arg A) error) (stop func(), err error) {
	if b == nil || !validName(name) || handle == nil {
		return nil, ErrInvalid
	}
	a := &action{run: func(ctx context.Context, raw json.RawMessage) error {
		var arg A
		if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
			if err := json.Unmarshal(trimmed, &arg); err != nil {
				return fmt.Errorf("%w: action %q: %v", ErrInvalid, name, err)
			}
		}
		return handle(ctx, arg)
	}}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, ErrClosed
	}
	if _, taken := b.actions[name]; taken {
		return nil, ErrExists
	}
	b.actions[name] = a
	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			if b.actions[name] == a {
				delete(b.actions, name)
			}
		})
	}, nil
}

// Get returns the current frame of the state exposed under name, as the bound service's Get does for a page, after the AllowRead check of ctx.
func (b *Bridge) Get(ctx context.Context, name string) (Frame, error) {
	e, err := b.state(ctx, name)
	if err != nil {
		return Frame{}, err
	}
	version, raw, err := e.read()
	if err != nil {
		return Frame{}, err
	}
	return Frame{Name: name, Version: version, Value: raw}, nil
}

// Watch sends the frames of the state exposed under name to send until ctx ends, the exposure stops, the source closes or the bridge closes, as the bound service's Watch does through a page's channel. The first frame is the current one, unless its version is after, which the caller already holds; versions count per source, so for a name exposed more than once, which may hold another source than the one the caller's version came from, the first frame is always sent. After that it sends a frame whenever the source publishes and the encoded projection differs from the last one sent; while send blocks, changes coalesce into the latest. It returns nil when ctx ends or send fails with mygo.ErrChannelClosed, which is how a page stops watching, and ErrClosed when the state goes away.
func (b *Bridge) Watch(ctx context.Context, name string, after uint64, send func(Frame) error) error {
	e, err := b.state(ctx, name)
	if err != nil {
		return err
	}
	changed := make(chan struct{}, 1)
	// Subscribe before the first read, so no change between them is lost.
	unsubscribe := e.subscribe(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	defer unsubscribe()
	var last json.RawMessage
	first := true
	for {
		version, raw, err := e.read()
		if err != nil {
			return err
		}
		skip := (!first && bytes.Equal(raw, last)) || (first && after != 0 && version == after && !e.reused)
		if !skip {
			if err := send(Frame{Name: name, Version: version, Value: raw}); err != nil {
				if errors.Is(err, mygo.ErrChannelClosed) {
					return nil
				}
				return err
			}
		}
		// Whether or not it was sent, the page holds a value equal to this one from here on.
		last, first = raw, false
		select {
		case <-changed:
		case <-ctx.Done():
			return nil
		case <-e.stopped:
			return ErrClosed
		case <-e.sourceDone:
			return ErrClosed
		}
	}
}

// Dispatch runs the action registered under name with its JSON argument, as the bound service's Dispatch does for a page, after the AllowDispatch check of ctx.
func (b *Bridge) Dispatch(ctx context.Context, name string, arg json.RawMessage) error {
	b.mu.Lock()
	closed := b.closed
	a, ok := b.actions[name]
	b.mu.Unlock()
	switch {
	case closed:
		return ErrClosed
	// The policy comes before the lookup, so a refused page cannot tell a registered name from a missing one.
	case b.options.AllowDispatch != nil && !b.options.AllowDispatch(ctx, name):
		return ErrDenied
	case !ok:
		return fmt.Errorf("%w: action %q", ErrNotFound, name)
	}
	return a.run(ctx, arg)
}

func (b *Bridge) state(ctx context.Context, name string) (*exposure, error) {
	b.mu.Lock()
	closed := b.closed
	e, ok := b.states[name]
	b.mu.Unlock()
	switch {
	case closed:
		return nil, ErrClosed
	// The policy comes before the lookup, so a refused page cannot tell an exposed name from a missing one.
	case b.options.AllowRead != nil && !b.options.AllowRead(ctx, name):
		return nil, ErrDenied
	case !ok:
		return nil, fmt.Errorf("%w: state %q", ErrNotFound, name)
	}
	return e, nil
}

func validName(name string) bool { return strings.TrimSpace(name) == name && name != "" }

// isNil reports whether an interface holds nothing or a nil pointer, map, slice, channel or function.
func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	default:
		return false
	}
}

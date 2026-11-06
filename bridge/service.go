package bridge

import (
	"context"
	"encoding/json"

	"github.com/egoist/mygo"
)

// service is the value Bind binds: its exported methods are what pages call, and nothing else of the Bridge is reachable from them.
type service struct {
	bridge *Bridge
}

// Get returns the current frame of a state.
func (s *service) Get(ctx context.Context, name string) (Frame, error) {
	return s.bridge.Get(ctx, name)
}

// Watch streams the frames of a state to the page until the page closes the channel, navigates away or closes its window, or the state goes away. after is the version the page already holds, or 0.
func (s *service) Watch(ctx context.Context, name string, after uint64, frames *mygo.Channel[Frame]) error {
	return s.bridge.Watch(ctx, name, after, frames.Send)
}

// Dispatch runs a registered action with its argument.
func (s *service) Dispatch(ctx context.Context, name string, arg json.RawMessage) error {
	return s.bridge.Dispatch(ctx, name, arg)
}

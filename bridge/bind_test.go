package bridge_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo"
	"github.com/yldm-tech/nodra/bridge"
)

// MyGo's services are bound for the whole process, and a name can be bound once, so the test binds once however often it runs.
var bindOnce sync.Once

// Binding goes through MyGo's checks of a service's methods, and the client mygo generate writes for an app that binds the bridge must type its calls.
func TestBindGeneratesTheClientsCalls(t *testing.T) {
	bindOnce.Do(func() { bridge.New(bridge.Options{}).BindAs("NodraBindTest") })
	src, err := mygo.GenerateTypeScript()
	if err != nil {
		t.Fatal(err)
	}
	client := string(src)
	for _, want := range []string{
		`call("NodraBindTest.Get", name)`,
		`call("NodraBindTest.Watch", name, after, frames)`,
		`call("NodraBindTest.Dispatch", name, arg)`,
		"watch(name: string, after: number, frames: Channel<Frame>): Promise<void>",
	} {
		if !strings.Contains(client, want) {
			t.Errorf("the generated client lacks %s:\n%s", want, client)
		}
	}
}

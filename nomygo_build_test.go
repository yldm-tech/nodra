package nodra_test

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoMyGoBuildTagDropsMyGo checks the nodra_nomygo opt-out: a program that uses only the core and DevTools must not link MyGo or purego, so a CGO_ENABLED=0 Linux build is a static executable that runs in scratch, distroless-static and musl images.
func TestNoMyGoBuildTagDropsMyGo(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes the go command")
	}
	goCmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command not found")
	}
	env := append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64", "GOFLAGS=", "GOWORK=off")

	list := exec.Command(goCmd, "list", "-deps", "-tags", "nodra_nomygo", ".", "./devtools", "./examples/basic")
	list.Env = env
	out, err := list.CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if strings.HasPrefix(pkg, "github.com/egoist/") || strings.HasPrefix(pkg, "github.com/ebitengine/") {
			t.Errorf("nodra_nomygo build depends on %s", pkg)
		}
	}

	binary := filepath.Join(t.TempDir(), "basic")
	build := exec.Command(goCmd, "build", "-tags", "nodra_nomygo", "-o", binary, "./examples/basic")
	build.Env = env
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	file, err := elf.Open(binary)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	for _, prog := range file.Progs {
		if prog.Type == elf.PT_INTERP {
			t.Fatal("nodra_nomygo CGO_ENABLED=0 linux build has a PT_INTERP header; it is dynamically linked")
		}
	}
}

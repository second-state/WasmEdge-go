//go:build windows

package wasmedge

import (
	"errors"
	"os"
	"testing"
)

func ownedWASIDiscardDescriptors(
	state *wasiModuleState,
) ([3]int32, [3]uint64) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.stdio.ownedDescriptors, state.stdio.expectedHandlers
}

func assertWASINullHandlersOpen(
	t *testing.T,
	handlers [3]uint64,
	want bool,
) {
	t.Helper()
	for i, handler := range handlers {
		if got := platformWASIStdioHandlerOpen(handler); got != want {
			t.Errorf("discard native handler %d (%d) open=%t, want %t",
				i, handler, got, want)
		}
	}
}

func TestWASIDiscardStdioOwnsCRTDescriptors(t *testing.T) {
	wasi, err := NewWASIModule(WASIConfig{Stdio: DiscardWASIStdio()})
	if err != nil {
		t.Fatal(err)
	}
	state := wasi.wasi
	descriptors, expectedHandlers := ownedWASIDiscardDescriptors(state)
	assertWASINullHandlersOpen(t, expectedHandlers, true)
	for fd := int32(0); fd < 3; fd++ {
		handler, err := wasi.WASINativeHandler(fd)
		if err != nil {
			t.Fatalf("discard fd %d: %v", fd, err)
		}
		if handler != expectedHandlers[fd] {
			t.Fatalf("discard fd %d handler = %d, want NUL handle %d",
				fd, handler, expectedHandlers[fd])
		}
	}

	if err := wasi.InitWASI(WASIConfig{
		Args:  []string{"second-run"},
		Stdio: DiscardWASIStdio(),
	}); err != nil {
		t.Fatalf("same-policy re-init: %v", err)
	}
	gotDescriptors, gotHandlers := ownedWASIDiscardDescriptors(state)
	if gotDescriptors != descriptors || gotHandlers != expectedHandlers {
		t.Fatalf(
			"re-init descriptors/handlers = %v/%v, want original %v/%v",
			gotDescriptors, gotHandlers, descriptors, expectedHandlers,
		)
	}
	assertWASINullHandlersOpen(t, expectedHandlers, true)

	if err := wasi.Close(); err != nil {
		t.Fatal(err)
	}
	assertWASINullHandlersOpen(t, expectedHandlers, false)
}

func TestWASIDiscardStdioReleasedOnVMReset(t *testing.T) {
	vm, err := NewVM(&Config{WASI: true})
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	wasi, ok := vm.WASIModule()
	if !ok {
		t.Fatal("WASI module missing")
	}
	if err := wasi.InitWASI(WASIConfig{Stdio: DiscardWASIStdio()}); err != nil {
		t.Fatal(err)
	}
	_, expectedHandlers := ownedWASIDiscardDescriptors(wasi.wasi)
	assertWASINullHandlersOpen(t, expectedHandlers, true)

	if err := vm.Reset(); err != nil {
		t.Fatal(err)
	}
	assertWASINullHandlersOpen(t, expectedHandlers, false)
}

func TestWASIStdioRejectsCustomDescriptorsOnWindows(t *testing.T) {
	files := [3]*os.File{}
	for i, suffix := range [...]string{"stdin", "stdout", "stderr"} {
		file, err := os.CreateTemp(t.TempDir(), suffix)
		if err != nil {
			t.Fatal(err)
		}
		files[i] = file
		t.Cleanup(func() { _ = file.Close() })
	}
	cfg := WASIConfig{
		Stdio: RedirectWASIStdio(files[0], files[1], files[2]),
	}

	if module, err := NewWASIModule(cfg); module != nil ||
		!errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("custom Windows stdio: module=%v error=%v", module, err)
	}

	wasi, err := NewWASIModule(WASIConfig{Stdio: InheritWASIStdio()})
	if err != nil {
		t.Fatal(err)
	}
	defer wasi.Close()
	if err := wasi.InitWASI(cfg); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("reconfigure custom Windows stdio: %v", err)
	}
}

func TestWASIStdioRejectsIncompleteSetOnWindows(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if module, err := NewWASIModule(WASIConfig{
		Stdio: RedirectWASIStdio(file, nil, nil),
	}); module != nil ||
		!errors.Is(err, ErrIncompleteWASIStdio) ||
		!errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("incomplete Windows stdio: module=%v error=%v", module, err)
	}
}

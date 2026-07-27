//go:build !windows

package wasmedge

import (
	"errors"
	"io"
	"os"
	"testing"
)

func newWASIStdioSet(t *testing.T, prefix string) [3]*os.File {
	t.Helper()
	var files [3]*os.File
	for i, suffix := range [...]string{"stdin", "stdout", "stderr"} {
		file, err := os.CreateTemp(t.TempDir(), prefix+"-"+suffix)
		if err != nil {
			t.Fatal(err)
		}
		files[i] = file
		t.Cleanup(func() { _ = file.Close() })
	}
	return files
}

func retainedWASIStdio(state *wasiModuleState) [3]*os.File {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.stdio.retained
}

func redirectedWASIStdio(files [3]*os.File) WASIStdio {
	return RedirectWASIStdio(files[0], files[1], files[2])
}

func ownedWASIDiscardResources(
	state *wasiModuleState,
) ([]*os.File, [3]int32) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return append([]*os.File(nil), state.stdio.owned...),
		state.stdio.ownedDescriptors
}

func assertWASIStdioFilesClosed(t *testing.T, files []*os.File) {
	t.Helper()
	for i, file := range files {
		if _, err := file.Stat(); !errors.Is(err, os.ErrClosed) {
			t.Errorf("owned discard file %d remains open: %v", i, err)
		}
	}
}

func TestWASIDiscardStdioOwnsNullFiles(t *testing.T) {
	wasi, err := NewWASIModule(WASIConfig{Stdio: DiscardWASIStdio()})
	if err != nil {
		t.Fatal(err)
	}
	state := wasi.wasi
	owned, descriptors := ownedWASIDiscardResources(state)
	if len(owned) != 3 {
		t.Fatalf("owned discard files = %d, want 3", len(owned))
	}
	for fd, file := range owned {
		handler, err := wasi.WASINativeHandler(int32(fd))
		if err != nil {
			t.Fatalf("discard fd %d: %v", fd, err)
		}
		if handler != uint64(file.Fd()) ||
			descriptors[fd] != int32(file.Fd()) {
			t.Fatalf(
				"discard fd %d: handler=%d descriptor=%d file=%d",
				fd, handler, descriptors[fd], file.Fd(),
			)
		}
	}
	buf := []byte{0xFF}
	if n, err := owned[0].Read(buf); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("discard stdin: n=%d err=%v, want EOF", n, err)
	}
	for fd := 1; fd <= 2; fd++ {
		if n, err := owned[fd].Write([]byte("discarded")); err != nil ||
			n != len("discarded") {
			t.Fatalf("discard output fd %d: n=%d err=%v", fd, n, err)
		}
	}

	if err := wasi.Close(); err != nil {
		t.Fatal(err)
	}
	assertWASIStdioFilesClosed(t, owned)
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
	owned, _ := ownedWASIDiscardResources(wasi.wasi)
	if len(owned) != 3 {
		t.Fatalf("owned discard files = %d, want 3", len(owned))
	}

	if err := vm.Reset(); err != nil {
		t.Fatal(err)
	}
	assertWASIStdioFilesClosed(t, owned)
}

func TestWASIStdioAndNativeHandlers(t *testing.T) {
	files := newWASIStdioSet(t, "native-handlers")
	in, out, errOut := files[0], files[1], files[2]

	wasi, err := NewWASIModule(WASIConfig{
		Stdio: RedirectWASIStdio(in, out, errOut),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wasi.Close()

	handler, err := wasi.WASINativeHandler(0)
	if err != nil {
		t.Fatal(err)
	}
	if handler != uint64(in.Fd()) {
		t.Fatalf("stdin native handler = %d, want %d", handler, in.Fd())
	}
	if _, err := wasi.WASINativeHandler(12345); !errors.Is(err, ErrWASINativeHandlerNotFound) {
		t.Fatalf("missing descriptor: %v", err)
	}
	partial := WASIConfig{Stdio: RedirectWASIStdio(in, nil, nil)}
	if err := wasi.InitWASI(partial); !errors.Is(err, ErrIncompleteWASIStdio) {
		t.Fatalf("partial stdio: %v", err)
	}
	if err := wasi.InitWASI(partial); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("partial stdio must wrap ErrInvalidArgument: %v", err)
	}

	if got, err := NewWASIModule(partial); got != nil ||
		!errors.Is(err, ErrIncompleteWASIStdio) ||
		!errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("partial standalone stdio: module=%v error=%v", got, err)
	}
}

func TestWASIStdioLifetimeFollowsModule(t *testing.T) {
	first := newWASIStdioSet(t, "first")
	second := newWASIStdioSet(t, "second")

	wasi, err := NewWASIModule(WASIConfig{
		Stdio: redirectedWASIStdio(first),
	})
	if err != nil {
		t.Fatal(err)
	}
	state := wasi.wasi
	if state == nil {
		t.Fatal("standalone WASI module has no provenance state")
	}
	if got := retainedWASIStdio(state); got != first {
		t.Fatalf("constructor stdio roots = %v, want %v", got, first)
	}

	if err := wasi.InitWASI(WASIConfig{
		Stdio: RedirectWASIStdio(second[0], nil, nil),
	}); !errors.Is(
		err,
		ErrIncompleteWASIStdio,
	) {
		t.Fatalf("failed re-init: %v", err)
	}
	if got := retainedWASIStdio(state); got != first {
		t.Fatalf("failed re-init replaced stdio roots = %v, want %v", got, first)
	}

	if err := wasi.InitWASI(WASIConfig{
		Args:  []string{"second-run"},
		Stdio: redirectedWASIStdio(first),
	}); err != nil {
		t.Fatalf("re-init with the same stdio: %v", err)
	}
	if got := retainedWASIStdio(state); got != first {
		t.Fatalf("same-mapping re-init changed stdio roots = %v, want %v", got, first)
	}

	if err := wasi.InitWASI(WASIConfig{
		Stdio: redirectedWASIStdio(second),
	}); !errors.Is(err, ErrWASIResourceMappingImmutable) ||
		!errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("changed stdio mapping: got %v, want immutable/unsupported", err)
	}
	if got := retainedWASIStdio(state); got != first {
		t.Fatalf("rejected replacement changed stdio roots = %v, want %v", got, first)
	}
	handler, err := wasi.WASINativeHandler(1)
	if err != nil {
		t.Fatal(err)
	}
	if handler != uint64(first[1].Fd()) {
		t.Fatalf("rejected replacement changed native stdout = %d, want %d",
			handler, first[1].Fd())
	}

	if err := wasi.InitWASI(WASIConfig{
		Stdio: InheritWASIStdio(),
	}); !errors.Is(
		err,
		ErrWASIResourceMappingImmutable,
	) {
		t.Fatalf("clearing initialized stdio: %v", err)
	}
	if err := wasi.Close(); err != nil {
		t.Fatal(err)
	}
	if got := retainedWASIStdio(state); got != ([3]*os.File{}) {
		t.Fatalf("Module.Close retained WASI stdio: %v", got)
	}
	if _, err := first[1].WriteString("caller still owns stdout"); err != nil {
		t.Fatalf("Module.Close closed caller-owned stdout: %v", err)
	}
}

func TestVMWASIStdioStateFollowsRegisteredGeneration(t *testing.T) {
	vm, err := NewVM(&Config{WASI: true})
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	firstView, ok := vm.WASIModule()
	if !ok {
		t.Fatal("first WASI view missing")
	}
	secondView, ok := vm.WASIModule()
	if !ok {
		t.Fatal("second WASI view missing")
	}
	if firstView.wasi == nil || firstView.wasi != secondView.wasi {
		t.Fatal("WASI views in one VM generation do not share state")
	}

	first := newWASIStdioSet(t, "vm-first")
	second := newWASIStdioSet(t, "vm-second")
	if err := firstView.InitWASI(WASIConfig{
		Stdio: redirectedWASIStdio(first),
	}); err != nil {
		t.Fatal(err)
	}
	if got := retainedWASIStdio(secondView.wasi); got != first {
		t.Fatalf("shared VM stdio roots = %v, want %v", got, first)
	}
	if err := secondView.InitWASI(WASIConfig{
		Stdio: redirectedWASIStdio(second),
	}); !errors.Is(err, ErrWASIResourceMappingImmutable) {
		t.Fatalf("changed VM stdio mapping: %v", err)
	}
	state := firstView.wasi
	if got := retainedWASIStdio(state); got != first {
		t.Fatalf("rejected VM replacement changed stdio roots = %v, want %v", got, first)
	}
	handler, err := secondView.WASINativeHandler(2)
	if err != nil {
		t.Fatal(err)
	}
	if handler != uint64(first[2].Fd()) {
		t.Fatalf("rejected VM replacement changed native stderr = %d, want %d",
			handler, first[2].Fd())
	}

	if err := vm.Reset(); err != nil {
		t.Fatal(err)
	}
	if got := retainedWASIStdio(state); got != ([3]*os.File{}) {
		t.Fatalf("VM.Reset retained WASI stdio: %v", got)
	}
}

func TestWASIPreopensCannotChangeAfterInitialization(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	wasi, err := NewWASIModule(WASIConfig{
		Preopens: []string{first},
		Stdio:    InheritWASIStdio(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wasi.Close()

	if err := wasi.InitWASI(WASIConfig{
		Args:     []string{"second-run"},
		Preopens: []string{first},
		Stdio:    InheritWASIStdio(),
	}); err != nil {
		t.Fatalf("same preopen mapping: %v", err)
	}
	if err := wasi.InitWASI(WASIConfig{
		Preopens: []string{second},
		Stdio:    InheritWASIStdio(),
	}); !errors.Is(err, ErrWASIResourceMappingImmutable) ||
		!errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("changed preopen mapping: got %v, want immutable/unsupported", err)
	}
}

func TestWASIStdioRejectsDescriptorOutsideInt32(t *testing.T) {
	invalid := os.NewFile(uintptr(1<<31), "outside-int32")
	if invalid == nil {
		t.Fatal("os.NewFile returned nil")
	}
	defer invalid.Close()
	cfg := WASIConfig{Stdio: RedirectWASIStdio(invalid, invalid, invalid)}
	if _, err := NewWASIModule(cfg); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("oversized file descriptor: %v", err)
	}
}

func TestWASIStdioRejectsClosedDescriptor(t *testing.T) {
	invalid := os.NewFile(uintptr(1<<30), "closed-or-unallocated")
	if invalid == nil {
		t.Fatal("os.NewFile returned nil")
	}
	defer invalid.Close()
	cfg := WASIConfig{Stdio: RedirectWASIStdio(invalid, invalid, invalid)}
	if _, err := NewWASIModule(cfg); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("closed file descriptor: %v", err)
	}

	first := newWASIStdioSet(t, "valid-before-failed-init")
	wasi, err := NewWASIModule(WASIConfig{
		Stdio: redirectedWASIStdio(first),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wasi.Close()
	if err := wasi.InitWASI(cfg); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("InitWASI closed file descriptor: %v", err)
	}
	if got := retainedWASIStdio(wasi.wasi); got != first {
		t.Fatalf("failed InitWASI replaced stdio roots: got %v, want %v", got, first)
	}
}

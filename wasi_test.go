package wasmedge

import (
	"errors"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestWASIExitCodeViaVM(t *testing.T) {
	vm, err := NewVM(&Config{WASI: true})
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	wasi, ok := vm.WASIModule()
	if !ok {
		t.Fatal("Config.WASI did not surface a WASI module")
	}
	if _, err := wasi.WASINativeHandler(0); !errors.Is(
		err,
		ErrWASINativeHandlerNotFound,
	) {
		t.Fatalf("uninitialized VM WASI module inherited stdin: %v", err)
	}

	// proc_exit(7) terminates gracefully: the run reports success and the
	// exit code is on the WASI module.
	if _, err := vm.RunBytes(testwasm.ProcExitModule(), "_start"); err != nil {
		t.Fatalf("proc_exit must surface as success, got %v", err)
	}
	if got, err := wasi.WASIExitCode(); err != nil || got != 7 {
		t.Fatalf("exit code: got %d, err %v, want 7", got, err)
	}
}

func TestWASIModuleStandalone(t *testing.T) {
	wasi, err := NewWASIModule(WASIConfig{
		Args:  []string{"prog", "arg1"},
		Envs:  []string{"K=V"},
		Stdio: InheritWASIStdio(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wasi.Close()

	// The module exposes the WASI preview1 surface.
	if _, ok := wasi.Function("proc_exit"); !ok {
		t.Fatal("proc_exit missing from standalone WASI module")
	}
	if got, err := wasi.WASIExitCode(); err != nil || got != 0 {
		t.Fatalf("fresh exit code: got %d, err %v", got, err)
	}

	// Wire it into an explicit pipeline and run the fixture.
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()
	store := NewStore()
	defer store.Close()
	if err := exec.RegisterImport(store, wasi); err != nil {
		t.Fatal(err)
	}
	registeredWASI, ok := store.Module("wasi_snapshot_preview1")
	if !ok {
		t.Fatal("registered standalone WASI module missing from Store")
	}
	if _, err := registeredWASI.WASIExitCode(); err != nil {
		t.Fatalf("borrowed Store alias lost WASI provenance: %v", err)
	}

	loader, _ := NewLoader(nil)
	defer loader.Close()
	ast, err := loader.LoadBytes(testwasm.ProcExitModule())
	if err != nil {
		t.Fatal(err)
	}
	defer ast.Close()
	validator, _ := NewValidator(nil)
	defer validator.Close()
	if err := validator.Validate(ast); err != nil {
		t.Fatal(err)
	}
	inst, err := exec.Instantiate(store, ast)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()

	fn, _ := inst.Function("_start")
	if _, err := exec.Invoke(fn); err != nil {
		t.Fatalf("proc_exit must surface as success, got %v", err)
	}
	if got, err := wasi.WASIExitCode(); err != nil || got != 7 {
		t.Fatalf("exit code: got %d, err %v, want 7", got, err)
	}
	if err := wasi.InitWASI(WASIConfig{
		Args:  []string{"second-run"},
		Stdio: InheritWASIStdio(),
	}); err != nil {
		t.Fatal(err)
	}
	if got, err := wasi.WASIExitCode(); err != nil || got != 0 {
		t.Fatalf("re-init exit code: got %d, err %v, want 0", got, err)
	}
}

func TestWASIInitializationRequiresExplicitStdio(t *testing.T) {
	if module, err := NewWASIModule(WASIConfig{}); module != nil ||
		!errors.Is(err, ErrWASIStdioPolicyRequired) ||
		!errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("implicit standalone stdio: module=%v error=%v", module, err)
	}

	vm, err := NewVM(&Config{WASI: true})
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	wasi, ok := vm.WASIModule()
	if !ok {
		t.Fatal("Config.WASI did not surface a WASI module")
	}
	if err := wasi.InitWASI(WASIConfig{}); !errors.Is(
		err,
		ErrWASIStdioPolicyRequired,
	) {
		t.Fatalf("implicit VM stdio: %v", err)
	}
	if _, err := wasi.WASINativeHandler(0); !errors.Is(
		err,
		ErrWASINativeHandlerNotFound,
	) {
		t.Fatalf("rejected initialization changed stdin mapping: %v", err)
	}
	if err := wasi.InitWASI(WASIConfig{Stdio: InheritWASIStdio()}); err != nil {
		t.Fatalf("explicit inherited stdio: %v", err)
	}
	if _, err := wasi.WASINativeHandler(0); err != nil {
		t.Fatalf("explicit inherited stdin is not mapped: %v", err)
	}

	if err := vm.Reset(); err != nil {
		t.Fatalf("reset VM: %v", err)
	}
	fresh, ok := vm.WASIModule()
	if !ok {
		t.Fatal("WASI module missing after Reset")
	}
	if _, err := fresh.WASINativeHandler(0); !errors.Is(
		err,
		ErrWASINativeHandlerNotFound,
	) {
		t.Fatalf("reset VM WASI module inherited stdin: %v", err)
	}
}

func TestWASIDiscardStdioKeepsMappingAcrossReinit(t *testing.T) {
	wasi, err := NewWASIModule(WASIConfig{
		Args:  []string{"first-run"},
		Stdio: DiscardWASIStdio(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wasi.Close()

	var before [3]uint64
	for fd := range before {
		before[fd], err = wasi.WASINativeHandler(int32(fd))
		if err != nil {
			t.Fatalf("discard fd %d: %v", fd, err)
		}
	}
	if err := wasi.InitWASI(WASIConfig{
		Args:  []string{"second-run"},
		Envs:  []string{"MODE=discard"},
		Stdio: DiscardWASIStdio(),
	}); err != nil {
		t.Fatalf("same-policy re-init: %v", err)
	}
	for fd, want := range before {
		got, err := wasi.WASINativeHandler(int32(fd))
		if err != nil {
			t.Fatalf("reinitialized discard fd %d: %v", fd, err)
		}
		if got != want {
			t.Fatalf("reinitialized discard fd %d = %d, want original %d",
				fd, got, want)
		}
	}
	if err := wasi.InitWASI(WASIConfig{
		Stdio: InheritWASIStdio(),
	}); !errors.Is(err, ErrWASIResourceMappingImmutable) {
		t.Fatalf("discard-to-inherit re-init: %v", err)
	}
}

func TestWASIDiscardReinitRestoresGuestClosedFD(t *testing.T) {
	vm, err := NewVM(&Config{WASI: true})
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	wasi, ok := vm.WASIModule()
	if !ok {
		t.Fatal("WASI module missing")
	}
	cfg := WASIConfig{Stdio: DiscardWASIStdio()}
	if err := wasi.InitWASI(cfg); err != nil {
		t.Fatal(err)
	}
	before, err := wasi.WASINativeHandler(0)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := vm.RunBytes(
		testwasm.WASICloseStdinModule(),
		"_start",
	); err != nil {
		t.Fatalf("guest fd_close: %v", err)
	}
	if _, err := wasi.WASINativeHandler(0); !errors.Is(
		err,
		ErrWASINativeHandlerNotFound,
	) {
		t.Fatalf("guest fd_close left fd 0 mapped: %v", err)
	}
	if err := wasi.InitWASI(cfg); err != nil {
		t.Fatalf("re-init after guest fd_close: %v", err)
	}
	after, err := wasi.WASINativeHandler(0)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("restored stdin handler = %d, want original %d", after, before)
	}
}

func TestWASIInitReleasesUncommittedStdioResources(t *testing.T) {
	t.Run("native mapping failure", func(t *testing.T) {
		var releases int
		state := newWASIModuleState(
			wasiStdioResources{},
			nil,
			false,
		)
		err := state.applyInit(
			nil,
			DiscardWASIStdio(),
			func() (
				wasiStdioDescriptors,
				wasiStdioResources,
				error,
			) {
				return wasiStdioDescriptors{custom: true},
					wasiStdioResources{
						mode:         wasiStdioDiscard,
						releaseOwned: func() { releases++ },
					}, nil
			},
			func(wasiStdioDescriptors) error { return ErrUnavailable },
		)
		if !errors.Is(err, ErrUnavailable) {
			t.Fatalf("native mapping failure: %v", err)
		}
		if releases != 1 {
			t.Fatalf("releases=%d, want 1", releases)
		}
		state.mu.Lock()
		initialized := state.initialized
		state.mu.Unlock()
		if initialized {
			t.Fatal("failed native mapping committed WASI state")
		}
	})

	t.Run("discard re-init reuses owned descriptors", func(t *testing.T) {
		var releases int
		want := [3]int32{41, 42, 43}
		state := newWASIModuleState(
			wasiStdioResources{
				mode:             wasiStdioDiscard,
				ownedDescriptors: want,
				releaseOwned:     func() { releases++ },
			},
			nil,
			true,
		)
		nativeCalls := 0
		err := state.applyInit(
			nil,
			DiscardWASIStdio(),
			func() (
				wasiStdioDescriptors,
				wasiStdioResources,
				error,
			) {
				t.Fatal("discard re-init allocated replacement descriptors")
				return wasiStdioDescriptors{}, wasiStdioResources{}, nil
			},
			func(stdio wasiStdioDescriptors) error {
				nativeCalls++
				got := [3]int32{stdio.stdin, stdio.stdout, stdio.stderr}
				if !stdio.custom || got != want {
					t.Fatalf("re-init descriptors = %v (custom=%t), want %v",
						got, stdio.custom, want)
				}
				return nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		if nativeCalls != 1 || releases != 0 {
			t.Fatalf("native calls=%d releases=%d, want 1 and 0",
				nativeCalls, releases)
		}
		state.close()
		if releases != 1 {
			t.Fatalf("teardown releases=%d, want 1", releases)
		}
	})

	t.Run("immutable mapping rejection avoids allocation", func(t *testing.T) {
		state := newWASIModuleState(
			wasiStdioResources{mode: wasiStdioInherit},
			nil,
			true,
		)
		err := state.applyInit(
			nil,
			DiscardWASIStdio(),
			func() (
				wasiStdioDescriptors,
				wasiStdioResources,
				error,
			) {
				t.Fatal("immutable mapping allocated discard descriptors")
				return wasiStdioDescriptors{}, wasiStdioResources{}, nil
			},
			func(wasiStdioDescriptors) error {
				t.Fatal("immutable mapping entered native code")
				return nil
			},
		)
		if !errors.Is(err, ErrWASIResourceMappingImmutable) {
			t.Fatalf("mapping rejection: %v", err)
		}
	})
}

func TestWASIProvenanceRejectsSpoofedModule(t *testing.T) {
	ordinary := NewModule("wasi_snapshot_preview1")
	defer ordinary.Close()
	if err := ordinary.InitWASI(WASIConfig{}); !errors.Is(err, ErrNotWASIModule) {
		t.Fatalf("InitWASI on spoofed-name ordinary module: %v", err)
	}
	if _, err := ordinary.WASINativeHandler(0); !errors.Is(err, ErrNotWASIModule) {
		t.Fatalf("WASINativeHandler on spoofed-name ordinary module: %v", err)
	}
	if _, err := ordinary.WASIExitCode(); !errors.Is(err, ErrNotWASIModule) {
		t.Fatalf("WASIExitCode on spoofed-name ordinary module: %v", err)
	}

	executor, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	store := NewStore()
	defer store.Close()
	if err := executor.RegisterImport(store, ordinary); err != nil {
		t.Fatal(err)
	}
	spoofedView, ok := store.Module("wasi_snapshot_preview1")
	if !ok {
		t.Fatal("spoofed-name module missing from Store")
	}
	if _, err := spoofedView.WASIExitCode(); !errors.Is(err, ErrNotWASIModule) {
		t.Fatalf("borrowed spoofed-name module gained WASI provenance: %v", err)
	}
}

func TestPluginListing(t *testing.T) {
	// Plugin availability depends on the host; this only asserts the calls
	// are safe and consistent with each other.
	names := PluginNames()
	for _, n := range names {
		p, ok := FindPlugin(n)
		if !ok {
			t.Fatalf("listed plugin %q not findable", n)
		}
		if p.Name() != n {
			t.Fatalf("plugin name mismatch: %q vs %q", p.Name(), n)
		}
	}
	if _, ok := FindPlugin("definitely-not-a-plugin"); ok {
		t.Fatal("ghost plugin found")
	}
}

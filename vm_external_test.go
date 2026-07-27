package wasmedge

import (
	"errors"
	"testing"
)

func TestVMExecuteRegisteredLeasesExternalStoreModule(t *testing.T) {
	started := make(chan struct{})
	unblock := make(chan struct{})
	env := NewModule("env")
	if err := env.AddFunction("host_add", MustWrapFunc(
		func(a, b int32) int32 {
			close(started)
			<-unblock
			return a + b
		},
	)); err != nil {
		t.Fatal(err)
	}

	executor, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	if err := executor.RegisterImport(store, env); err != nil {
		t.Fatal(err)
	}
	vm, err := NewVM(nil, WithExternalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = vm.Close()
		_ = env.Close()
		_ = store.Close()
		_ = executor.Close()
	})

	view, ok := vm.RegisteredModule("env")
	if !ok {
		t.Fatal("VM did not expose the pre-existing external-store module")
	}
	execution, err := vm.ExecuteRegisteredAsync("env", "host_add", I32(20), I32(22))
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := env.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("close module during registered execution: got %v, want ErrInUse", err)
	}
	close(unblock)
	out, err := execution.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].I32() != 42 {
		t.Fatalf("registered execution result: %v", out)
	}
	if err := env.Close(); err != nil {
		t.Fatalf("close module after registered execution: %v", err)
	}
	assertVMPanics(t, "registered view after external module Close", func() {
		_ = view.Name()
	})
}

func TestVMClosePreservesCallerOwnedExternalStoreRegistrations(t *testing.T) {
	env := NewModule("env")
	executor, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	if err := executor.RegisterImport(store, env); err != nil {
		t.Fatal(err)
	}
	vm, err := NewVM(nil, WithExternalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = vm.Close()
		_ = env.Close()
		_ = store.Close()
		_ = executor.Close()
	})

	if err := vm.RegisterImportWithAlias(env, "vm_alias"); err != nil {
		t.Fatal(err)
	}
	if err := vm.Close(); err != nil {
		t.Fatal(err)
	}
	alias, ok := store.Module("vm_alias")
	if !ok {
		t.Fatal("VM.Close removed its caller-owned import alias")
	}
	view, ok := store.Module("env")
	if !ok {
		t.Fatal("VM.Close removed the pre-existing registration")
	}
	if got, aliasGot := view.Name(), alias.Name(); got != "env" || aliasGot != "env" {
		t.Fatalf("external module view names: direct=%q alias=%q", got, aliasGot)
	}
	if len(store.state.retained) != 2 ||
		store.state.retained[0] != env ||
		store.state.retained[1] != env {
		t.Fatalf("VM.Close removed the wrong Go retention edge: %#v", store.state.retained)
	}
	if err := env.Close(); err != nil {
		t.Fatalf("close imported module after VM.Close: %v", err)
	}
	if _, ok := store.Module("env"); ok {
		t.Fatal("closed module remained directly registered")
	}
	if _, ok := store.Module("vm_alias"); ok {
		t.Fatal("closed module alias remained registered")
	}
	assertVMPanics(t, "direct view after module Close", func() { _ = view.Name() })
	assertVMPanics(t, "alias view after module Close", func() { _ = alias.Name() })
}

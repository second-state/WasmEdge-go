package wasmedge

import (
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestStoreModuleViewExpiresWithOwnedModule(t *testing.T) {
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()

	store := NewStore()
	defer store.Close()
	owned := NewModule("owned")
	if err := exec.RegisterImport(store, owned); err != nil {
		t.Fatal(err)
	}
	if err := exec.RegisterImportWithAlias(store, owned, "alias"); err != nil {
		t.Fatal(err)
	}

	view, ok := store.Module("owned")
	if !ok {
		t.Fatal("registered module view not found")
	}
	if got := view.Name(); got != "owned" {
		t.Fatalf("view name: got %q, want owned", got)
	}
	if err := owned.Close(); err != nil {
		t.Fatalf("close registered module: %v", err)
	}
	if _, ok := store.Module("owned"); ok {
		t.Fatal("closed module remained registered in Store")
	}
	if _, ok := store.Module("alias"); ok {
		t.Fatal("closed module alias remained registered in Store")
	}
	if len(store.state.retained) != 0 {
		t.Fatalf("closed module remained in Go Store state: %#v", store.state.retained)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("borrowed store view remained usable after owned module Close")
		}
	}()
	_ = view.Name()
}

func TestBorrowedVMStoreCloseKeepsSharedState(t *testing.T) {
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}

	store := vm.Store()
	retained := append([]*Module(nil), vm.storeState.retained...)
	if len(retained) == 0 {
		t.Fatal("VM registration was not retained")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close borrowed Store view: %v", err)
	}
	if len(vm.storeState.retained) != len(retained) {
		t.Fatalf("borrowed Store.Close released shared state: got %d retained modules, want %d",
			len(vm.storeState.retained), len(retained))
	}
	for i := range retained {
		if vm.storeState.retained[i] != retained[i] {
			t.Fatalf("borrowed Store.Close changed retained module %d", i)
		}
	}
	if _, ok := store.Module("calc"); !ok {
		t.Fatal("borrowed Store view became unusable after Close")
	}
	out, err := vm.ExecuteRegistered("calc", "add", I32(20), I32(22))
	if err != nil || len(out) != 1 || out[0].I32() != 42 {
		t.Fatalf("registration unusable after borrowed Store.Close: out=%v err=%v", out, err)
	}
}

func TestStoreStateForgetLastRegistration(t *testing.T) {
	first := &Module{}
	second := &Module{}
	backing := []*Module{first, second, first, first, second}
	state := &storeState{retained: backing}

	state.forgetLast([]*Module{first, second, first})
	if got := state.retained; len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("forgetLast did not preserve earlier registrations: %#v", got)
	}
	for i := len(state.retained); i < len(backing); i++ {
		if backing[i] != nil {
			t.Fatalf("forgetLast retained stale tail entry %d", i)
		}
	}

	state.forget([]*Module{first})
	if got := state.retained; len(got) != 1 || got[0] != second {
		t.Fatalf("forget did not remove all matching registrations: %#v", got)
	}
	for i := len(state.retained); i < len(backing); i++ {
		if backing[i] != nil {
			t.Fatalf("forget retained stale tail entry %d", i)
		}
	}
}

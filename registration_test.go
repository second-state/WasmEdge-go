package wasmedge

import (
	"errors"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestRegisterImportRejectsBorrowedModule(t *testing.T) {
	source, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := source.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	borrowed, ok := source.RegisteredModule("calc")
	if !ok {
		t.Fatal("source registered module missing")
	}

	executor, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	store := NewStore()
	defer store.Close()
	if err := executor.RegisterImport(store, borrowed); !errors.Is(err, ErrOwnership) {
		t.Fatalf("Executor.RegisterImport: got %v, want ErrOwnership", err)
	}
	if err := executor.RegisterImportWithAlias(
		store, borrowed, "alias",
	); !errors.Is(err, ErrOwnership) {
		t.Fatalf("Executor.RegisterImportWithAlias: got %v, want ErrOwnership", err)
	}
	if len(store.state.retained) != 0 {
		t.Fatalf("rejected Executor registration changed Store state: %#v", store.state.retained)
	}

	target, err := NewVM(nil, WithExternalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := target.RegisterImport(borrowed); !errors.Is(err, ErrOwnership) {
		t.Fatalf("VM.RegisterImport: got %v, want ErrOwnership", err)
	}
	if err := target.RegisterImportWithAlias(
		borrowed, "alias",
	); !errors.Is(err, ErrOwnership) {
		t.Fatalf("VM.RegisterImportWithAlias: got %v, want ErrOwnership", err)
	}
	if len(store.state.retained) != 0 {
		t.Fatalf("rejected VM registration changed Store state: %#v", store.state.retained)
	}
}

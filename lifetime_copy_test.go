package wasmedge

import (
	"errors"
	"testing"
)

func TestMemoryValueCopySharesLifetime(t *testing.T) {
	memory, err := NewMemory(MemoryType{Limits: Limits{Min: 1}})
	if err != nil {
		t.Fatal(err)
	}
	copied := *memory

	release, err := memory.acquireLease()
	if err != nil {
		t.Fatal(err)
	}
	if err := copied.Close(); !errors.Is(err, ErrInUse) {
		release()
		_ = memory.Close()
		t.Fatalf("close copied Memory while original is leased: %v", err)
	}
	if !memory.life.alive() || !copied.life.alive() {
		release()
		t.Fatal("failed Close changed shared lifetime state")
	}
	release()

	if err := copied.Close(); err != nil {
		t.Fatal(err)
	}
	if memory.life.alive() {
		// Do not call memory.Close on this failure path: an implementation with
		// per-copy state already deleted the shared native pointer above.
		t.Fatal("closing a copied Memory did not close the original wrapper")
	}
	if err := memory.Close(); err != nil {
		t.Fatalf("close original after copied wrapper: %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("original Memory remained usable after copied wrapper closed it")
		}
	}()
	_ = memory.PageCount()
}

func TestFunctionValueCopySharesTransfer(t *testing.T) {
	function := MustWrapFunc(func() int32 { return 42 })
	copied := *function
	token := function.token

	module := NewModule("copied-function")
	defer module.Close()
	if err := module.AddFunction("answer", function); err != nil {
		_ = function.Close()
		t.Fatal(err)
	}
	if copied.life.alive() {
		// The module now owns the native function. Avoid closing the stale copy
		// on this failure path because a per-copy lifetime would delete it.
		t.Fatal("transferring the original Function did not disarm its copy")
	}
	if err := copied.Close(); err != nil {
		t.Fatalf("close copied Function after transfer: %v", err)
	}
	if _, ok := hostFuncTokens.Load(token); !ok {
		t.Fatal("closing a copied transferred Function released the module-owned callback")
	}

	view, ok := module.Function("answer")
	if !ok {
		t.Fatal("transferred function is missing")
	}
	executor, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	out, err := executor.Invoke(view)
	if err != nil {
		t.Fatalf("invoke after closing copied transferred wrapper: %v", err)
	}
	if len(out) != 1 || out[0].I32() != 42 {
		t.Fatalf("invoke result: %v", out)
	}

	if err := module.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := hostFuncTokens.Load(token); ok {
		t.Fatal("module close did not release copied function callback")
	}
}

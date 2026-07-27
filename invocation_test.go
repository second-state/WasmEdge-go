package wasmedge

import (
	"errors"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func instantiatedAddVM(t *testing.T) *VM {
	t.Helper()
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = vm.Close() })
	if err := vm.LoadBytes(testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatal(err)
	}
	return vm
}

func requireErrorCode(t *testing.T, err error, code ErrCode) {
	t.Helper()
	var native *Error
	if !errors.As(err, &native) || native.Code != code {
		t.Fatalf("error = %v, want WasmEdge code %s", err, code)
	}
}

func TestVMRejectedInvocationDoesNotRootReference(t *testing.T) {
	vm := instantiatedAddVM(t)
	if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}

	t.Run("missing synchronous function", func(t *testing.T) {
		ref := NewExternRef("missing-sync")
		_, err := vm.Execute("missing", ExternRefValue(ref))
		requireErrorCode(t, err, ErrCodeFuncNotFound)
		if err := ref.Close(); err != nil {
			t.Fatalf("rejected reference remained rooted: %v", err)
		}
	})

	t.Run("wrong synchronous signature", func(t *testing.T) {
		ref := NewExternRef("signature-sync")
		_, err := vm.Execute("add", ExternRefValue(ref))
		requireErrorCode(t, err, ErrCodeFuncSigMismatch)
		if err := ref.Close(); err != nil {
			t.Fatalf("rejected reference remained rooted: %v", err)
		}
	})

	t.Run("missing asynchronous function", func(t *testing.T) {
		ref := NewExternRef("missing-async")
		execution, err := vm.ExecuteAsync("missing", ExternRefValue(ref))
		if execution != nil {
			_ = execution.Close()
			t.Fatal("missing function returned an Execution")
		}
		requireErrorCode(t, err, ErrCodeFuncNotFound)
		if err := ref.Close(); err != nil {
			t.Fatalf("rejected reference remained rooted: %v", err)
		}
	})

	t.Run("wrong asynchronous signature", func(t *testing.T) {
		ref := NewExternRef("signature-async")
		execution, err := vm.ExecuteAsync("add", ExternRefValue(ref))
		if execution != nil {
			_ = execution.Close()
			t.Fatal("wrong signature returned an Execution")
		}
		requireErrorCode(t, err, ErrCodeFuncSigMismatch)
		if err := ref.Close(); err != nil {
			t.Fatalf("rejected reference remained rooted: %v", err)
		}
	})

	t.Run("missing registered function", func(t *testing.T) {
		ref := NewExternRef("missing-registered")
		_, err := vm.ExecuteRegistered("calc", "missing", ExternRefValue(ref))
		requireErrorCode(t, err, ErrCodeFuncNotFound)
		if err := ref.Close(); err != nil {
			t.Fatalf("rejected reference remained rooted: %v", err)
		}
	})

	t.Run("wrong registered async signature", func(t *testing.T) {
		ref := NewExternRef("signature-registered-async")
		execution, err := vm.ExecuteRegisteredAsync(
			"calc",
			"add",
			ExternRefValue(ref),
		)
		if execution != nil {
			_ = execution.Close()
			t.Fatal("wrong signature returned an Execution")
		}
		requireErrorCode(t, err, ErrCodeFuncSigMismatch)
		if err := ref.Close(); err != nil {
			t.Fatalf("rejected reference remained rooted: %v", err)
		}
	})
}

func TestExecutorRejectedInvocationDoesNotRootReference(t *testing.T) {
	module := NewModule("host")
	t.Cleanup(func() { _ = module.Close() })
	owned := MustWrapFunc(func(int32) {})
	if err := module.AddFunction("consume", owned); err != nil {
		_ = owned.Close()
		t.Fatal(err)
	}
	function, ok := module.Function("consume")
	if !ok {
		t.Fatal("host function missing after transfer")
	}
	executor, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = executor.Close() })

	for _, async := range []bool{false, true} {
		name := "sync"
		if async {
			name = "async"
		}
		t.Run(name, func(t *testing.T) {
			ref := NewExternRef(name)
			var err error
			if async {
				var execution *Execution
				execution, err = executor.InvokeAsync(function, ExternRefValue(ref))
				if execution != nil {
					_ = execution.Close()
					t.Fatal("wrong signature returned an Execution")
				}
			} else {
				_, err = executor.Invoke(function, ExternRefValue(ref))
			}
			requireErrorCode(t, err, ErrCodeFuncSigMismatch)
			if err := ref.Close(); err != nil {
				t.Fatalf("rejected reference remained rooted: %v", err)
			}
		})
	}
}

func TestClassicReferenceKindMismatchDoesNotRootReference(t *testing.T) {
	module := NewModule("host")
	owned := MustWrapFunc(func(*Function) {})
	if err := module.AddFunction("consume_funcref", owned); err != nil {
		_ = owned.Close()
		_ = module.Close()
		t.Fatal(err)
	}
	function, ok := module.Function("consume_funcref")
	if !ok {
		_ = module.Close()
		t.Fatal("host function missing after transfer")
	}
	executor, err := NewExecutor(nil)
	if err != nil {
		_ = module.Close()
		t.Fatal(err)
	}
	defer executor.Close()
	defer module.Close()

	ref := NewExternRef("wrong classic reference kind")
	_, err = executor.Invoke(function, ExternRefValue(ref))
	requireErrorCode(t, err, ErrCodeFuncSigMismatch)
	if err := ref.Close(); err != nil {
		t.Fatalf("classic-kind mismatch remained rooted: %v", err)
	}
}

func TestInvocationReferencePreparationIsTransactional(t *testing.T) {
	module := NewModule("host")
	owned := MustWrapFunc(func(*ExternRef, *Function) {})
	if err := module.AddFunction("consume_refs", owned); err != nil {
		_ = owned.Close()
		_ = module.Close()
		t.Fatal(err)
	}
	function, ok := module.Function("consume_refs")
	if !ok {
		_ = module.Close()
		t.Fatal("host function missing after transfer")
	}
	executor, err := NewExecutor(nil)
	if err != nil {
		_ = module.Close()
		t.Fatal(err)
	}
	defer executor.Close()
	defer module.Close()

	call := newCallContext(nil, nil)
	callScoped := FuncRefValue(function)
	callScoped.owner = call
	ref := NewExternRef("transaction rollback")
	_, err = executor.Invoke(
		function,
		ExternRefValue(ref),
		callScoped,
	)
	call.expire()
	if !errors.Is(err, ErrOwnership) {
		t.Fatalf("transactional reference preparation: got %v, want ErrOwnership", err)
	}
	if err := ref.Close(); err != nil {
		t.Fatalf("failed preparation retained an earlier reference: %v", err)
	}
}

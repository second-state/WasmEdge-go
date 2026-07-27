package wasmedge

import (
	"errors"
	"sync"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestHostFunctionResultRejectsReferenceCycle(t *testing.T) {
	var first, second *Function
	first = MustWrapFunc(func() *Function { return second })
	second = MustWrapFunc(func() *Function { return first })
	executor, err := NewExecutor(nil)
	if err != nil {
		_ = first.Close()
		_ = second.Close()
		t.Fatal(err)
	}
	defer executor.Close()

	if _, err := executor.Invoke(first); err != nil {
		t.Fatalf("create first dependency edge: %v", err)
	}
	if _, err := executor.Invoke(second); !errors.Is(err, ErrReferenceCycle) {
		t.Fatalf("reverse dependency edge: got %v, want ErrReferenceCycle", err)
	}

	// Rejecting the reverse edge keeps the graph explicitly destructible:
	// closing the producer releases its lease on the second function.
	if err := first.Close(); err != nil {
		t.Fatalf("close first function: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("close second function after dependency release: %v", err)
	}
}

func TestHostFunctionAliasedResultsAvoidModuleCycles(t *testing.T) {
	var aliased Value
	producer, err := NewFunction(
		FunctionType{Results: []ValType{ValTypeFuncRef()}},
		func(*CallContext, []Value) ([]Value, error) {
			return []Value{aliased}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	env := NewModule("env")
	if err := env.AddFunction("p", producer); err != nil {
		_ = producer.Close()
		_ = env.Close()
		t.Fatal(err)
	}

	loader, err := NewLoader(nil)
	if err != nil {
		_ = env.Close()
		t.Fatal(err)
	}
	defer loader.Close()
	ast, err := loader.LoadBytes(testwasm.FuncRefAliasModule())
	if err != nil {
		_ = env.Close()
		t.Fatal(err)
	}
	defer ast.Close()
	validator, err := NewValidator(nil)
	if err != nil {
		_ = env.Close()
		t.Fatal(err)
	}
	defer validator.Close()
	if err := validator.Validate(ast); err != nil {
		_ = env.Close()
		t.Fatal(err)
	}

	executor, err := NewExecutor(nil)
	if err != nil {
		_ = env.Close()
		t.Fatal(err)
	}
	defer executor.Close()
	store := NewStore()
	defer store.Close()
	if err := executor.RegisterImport(store, env); err != nil {
		_ = env.Close()
		t.Fatal(err)
	}
	guest, err := executor.Instantiate(store, ast)
	if err != nil {
		_ = env.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = guest.Close()
		_ = env.Close()
	})

	getP, ok := guest.Function("get_p")
	if !ok {
		t.Fatal("get_p export missing")
	}
	results, err := executor.Invoke(getP)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Kind() != ValKindFuncRef {
		t.Fatalf("get_p results = %v, want one funcref", results)
	}
	aliased = results[0]

	borrowedProducer, ok := env.Function("p")
	if !ok {
		t.Fatal("p export missing")
	}
	if _, err := executor.Invoke(borrowedProducer); err != nil {
		t.Fatalf("return aliased self reference: %v", err)
	}

	// A different guest-owned function cannot be rooted by env.p: guest
	// already leases env as an import, so the reverse edge would be a cycle.
	aliased = FuncRefValue(getP)
	if _, err := executor.Invoke(borrowedProducer); !errors.Is(err, ErrReferenceCycle) {
		t.Fatalf("return imported guest reference: got %v, want ErrReferenceCycle", err)
	}

	// guest already leases env as an import dependency. The aliased result
	// must not make env's host callback lease guest back and form a cycle.
	if err := guest.Close(); err != nil {
		t.Fatalf("close guest after aliased self result: %v", err)
	}
	if err := env.Close(); err != nil {
		t.Fatalf("close host module after guest: %v", err)
	}
}

func TestHostFunctionRegistryPublishesInitializedEntries(t *testing.T) {
	target := MustWrapFunc(func() {})
	producer := MustWrapFunc(func() *Function { return target })
	executor, err := NewExecutor(nil)
	if err != nil {
		_ = producer.Close()
		_ = target.Close()
		t.Fatal(err)
	}
	defer executor.Close()

	const iterations = 300
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range iterations {
			transient, err := NewFunction(FunctionType{}, func(*CallContext, []Value) ([]Value, error) {
				return nil, nil
			})
			if err != nil {
				errs <- err
				return
			}
			if err := transient.Close(); err != nil {
				errs <- err
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for range iterations {
			if _, err := executor.Invoke(producer); err != nil {
				errs <- err
				return
			}
		}
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	if err := producer.Close(); err != nil {
		t.Fatalf("close producer: %v", err)
	}
	if err := target.Close(); err != nil {
		t.Fatalf("close target after producer root release: %v", err)
	}
}

func TestHostFunctionResultRejectsReverseVMImportDependency(t *testing.T) {
	var aliased Value
	producer, err := NewFunction(
		FunctionType{Results: []ValType{ValTypeFuncRef()}},
		func(*CallContext, []Value) ([]Value, error) {
			return []Value{aliased}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	env := NewModule("env")
	if err := env.AddFunction("p", producer); err != nil {
		_ = producer.Close()
		_ = env.Close()
		t.Fatal(err)
	}

	vm, err := NewVM(nil)
	if err != nil {
		_ = env.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = vm.Close()
		_ = env.Close()
	})
	// Register the guest before env so it has no native import dependency on
	// env. The later direct VM import lease is still enough to make a reverse
	// host-result root unsafe.
	if err := vm.RegisterModuleBytes("guest", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	if err := vm.RegisterImport(env); err != nil {
		t.Fatal(err)
	}
	guest, ok := vm.RegisteredModule("guest")
	if !ok {
		t.Fatal("registered guest module missing")
	}
	add, ok := guest.Function("add")
	if !ok {
		t.Fatal("registered guest add function missing")
	}
	aliased = FuncRefValue(add)

	if _, err := vm.ExecuteRegistered("env", "p"); !errors.Is(err, ErrReferenceCycle) {
		t.Fatalf("return VM-owned reference: got %v, want ErrReferenceCycle", err)
	}
	if err := vm.Close(); err != nil {
		t.Fatalf("close VM after rejected reverse edge: %v", err)
	}
	if err := env.Close(); err != nil {
		t.Fatalf("close imported module after VM: %v", err)
	}
}

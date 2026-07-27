package wasmedge

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

type panicIsHostError struct{}

func (panicIsHostError) Error() string { return "panic in Is" }
func (panicIsHostError) Is(error) bool { panic("panic from error.Is") }

type panicAsHostError struct{}

func (panicAsHostError) Error() string { return "panic in As" }
func (panicAsHostError) As(any) bool   { panic("panic from error.As") }

// instantiateWithHost loads the HostCall fixture, registers an "env" module
// providing host_add, and returns the instantiated module plus everything
// needed to invoke it. The cleanup order matters and doubles as lifecycle
// documentation: instance first, then store, then its imported env module.
func instantiateWithHost(t *testing.T, hostAdd *Function) (*Executor, *Module) {
	t.Helper()

	env := NewModule("env")
	if env == nil {
		t.Fatal("NewModule failed")
	}
	if err := env.AddFunction("host_add", hostAdd); err != nil {
		t.Fatal(err)
	}

	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { loader.Close() })

	ast, err := loader.LoadBytes(testwasm.HostCallModule())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ast.Close() })

	val, err := NewValidator(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { val.Close() })
	if err := val.Validate(ast); err != nil {
		t.Fatal(err)
	}

	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	if err := exec.RegisterImport(store, env); err != nil {
		t.Fatal(err)
	}
	inst, err := exec.Instantiate(store, ast)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		inst.Close()
		store.Close()
		env.Close()
		exec.Close()
	})
	return exec, inst
}

func invokeCallHost(t *testing.T, exec *Executor, inst *Module, a, b int32) ([]Value, error) {
	t.Helper()
	fn, ok := inst.Function("call_host")
	if !ok {
		t.Fatal("call_host export missing")
	}
	return exec.Invoke(fn, I32(a), I32(b))
}

func TestHostFunctionRoundTrip(t *testing.T) {
	var sawCtx bool
	add := mustNewHostFunction(t,
		FunctionType{
			Params:  []ValType{ValTypeI32(), ValTypeI32()},
			Results: []ValType{ValTypeI32()},
		},
		func(call *CallContext, params []Value) ([]Value, error) {
			sawCtx = call != nil && call.Executor() != nil && call.Module() != nil
			return []Value{I32(params[0].I32() + params[1].I32())}, nil
		})

	exec, inst := instantiateWithHost(t, add)
	out, err := invokeCallHost(t, exec, inst, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].I32() != 7 {
		t.Fatalf("out=%v", out)
	}
	if !sawCtx {
		t.Fatal("CallContext was not fully populated during the host call")
	}
}

func TestWrapFunc(t *testing.T) {
	add, err := WrapFunc(func(a, b int32) int32 { return a + b })
	if err != nil {
		t.Fatal(err)
	}
	exec, inst := instantiateWithHost(t, add)
	out, err := invokeCallHost(t, exec, inst, 20, 22)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].I32() != 42 {
		t.Fatalf("out=%v", out)
	}
}

func TestWrapFuncSignatures(t *testing.T) {
	// Signature derivation only; invocation is covered elsewhere.
	cases := []struct {
		fn       any
		nParams  int
		nResults int
	}{
		{func() {}, 0, 0},
		{func(int32) int64 { return 0 }, 1, 1},
		{func(*CallContext, uint64, float32) (float64, error) { return 0, nil }, 2, 1},
		{func(uint32) error { return nil }, 1, 0},
	}
	for i, c := range cases {
		f, err := WrapFunc(c.fn)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		ft := f.Type()
		if len(ft.Params) != c.nParams || len(ft.Results) != c.nResults {
			t.Errorf("case %d: got %d/%d params/results, want %d/%d",
				i, len(ft.Params), len(ft.Results), c.nParams, c.nResults)
		}
		f.Close()
	}

	if _, err := WrapFunc(42); err == nil {
		t.Error("non-func must be rejected")
	}
	if _, err := WrapFunc(func(string) {}); err == nil {
		t.Error("string parameter must be rejected")
	}
	if _, err := WrapFunc(func(...int32) {}); err == nil {
		t.Error("variadic func must be rejected")
	}
	if _, err := WrapFunc(func([15]byte) {}); err == nil {
		t.Error("[15]byte parameter must be rejected")
	}
	if _, err := WrapFunc(func(**ExternRef) {}); err == nil {
		t.Error("pointer types other than *ExternRef and *Function must be rejected")
	}
	var nilFunc func()
	if _, err := WrapFunc(nilFunc); err == nil {
		t.Error("nil func must be rejected")
	}
}

func TestWrapFuncKinds(t *testing.T) {
	lanes := [16]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	payload := &struct{ name string }{name: "externref"}
	ref := NewExternRef(payload)
	defer ref.Close()
	target := MustWrapFunc(func() {})
	defer target.Close()

	var receivedV128 [16]byte
	var receivedRef *ExternRef
	var receivedFunc *Function
	wrapped := MustWrapFunc(func(
		vector [16]byte,
		extern *ExternRef,
		function *Function,
	) ([16]byte, *ExternRef, *Function) {
		receivedV128, receivedRef, receivedFunc = vector, extern, function
		return vector, extern, function
	})
	defer wrapped.Close()

	ft := wrapped.Type()
	wantKinds := []ValKind{ValKindV128, ValKindExternRef, ValKindFuncRef}
	for label, types := range map[string][]ValType{
		"parameters": ft.Params,
		"results":    ft.Results,
	} {
		if len(types) != len(wantKinds) {
			t.Fatalf("%s: got %d kinds, want %d", label, len(types), len(wantKinds))
		}
		for i, typ := range types {
			if got := typ.Kind(); got != wantKinds[i] {
				t.Errorf("%s[%d]: got %s, want %s", label, i, got, wantKinds[i])
			}
		}
	}

	entry, ok := hostFuncEntryForToken(wrapped.token)
	if !ok {
		t.Fatal("wrapped function has corrupt host entry")
	}
	out, err := entry.fn(nil, []Value{
		V128(lanes),
		ExternRefValue(ref),
		FuncRefValue(target),
	})
	if err != nil {
		t.Fatal(err)
	}
	if receivedV128 != lanes {
		t.Fatalf("received v128=%x, want %x", receivedV128, lanes)
	}
	if receivedRef == nil || receivedRef.Value() != payload {
		t.Fatal("externref parameter lost its payload")
	}
	if receivedFunc == nil || receivedFunc.ptr != target.ptr {
		t.Fatal("funcref parameter lost its function")
	}
	if len(out) != 3 || out[0].V128() != lanes {
		t.Fatalf("v128 result=%v", out)
	}
	if out[1].owner == nil || out[1].ExternRef().Value() != payload {
		t.Fatal("externref result did not retain its owner")
	}
	if out[2].owner == nil || out[2].FuncRef().ptr != target.ptr {
		t.Fatal("funcref result did not retain its owner")
	}

	out, err = entry.fn(nil, []Value{V128(lanes), NullExternRef(), NullFuncRef()})
	if err != nil {
		t.Fatal(err)
	}
	if receivedRef != nil || receivedFunc != nil {
		t.Fatal("null WASM references must unpack as nil Go pointers")
	}
	if !out[1].IsNullRef() || !out[2].IsNullRef() {
		t.Fatal("nil Go pointers must pack as null WASM references")
	}

	badInputs := []struct {
		name string
		in   []Value
		want string
	}{
		{"v128", []Value{I32(0), NullExternRef(), NullFuncRef()}, "want v128"},
		{"externref", []Value{V128(lanes), NullFuncRef(), NullFuncRef()}, "want externref"},
		{"funcref", []Value{V128(lanes), NullExternRef(), NullExternRef()}, "want funcref"},
	}
	for _, tc := range badInputs {
		t.Run("reject_"+tc.name+"_mismatch", func(t *testing.T) {
			if _, err := entry.fn(nil, tc.in); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got error %v, want one containing %q", err, tc.want)
			}
		})
	}
	if _, err := entry.fn(nil, []Value{V128(lanes)}); err == nil {
		t.Fatal("wrong parameter arity must be rejected")
	}
}

func TestWrapFuncReferenceParameterLifetimes(t *testing.T) {
	ref := NewExternRef("payload")
	defer ref.Close()
	target := MustWrapFunc(func() {})
	defer target.Close()

	var escapedRef *ExternRef
	var escapedFunc *Function
	wrapped := MustWrapFunc(func(ref *ExternRef, fn *Function) {
		escapedRef, escapedFunc = ref, fn
	})
	defer wrapped.Close()
	entry, ok := hostFuncEntryForToken(wrapped.token)
	if !ok {
		t.Fatal("wrapped function has corrupt host entry")
	}

	externValue := ExternRefValue(ref)
	externValue.owner = nil // matches Values copied in from the C callback
	funcValue := FuncRefValue(target)
	funcValue.owner = nil
	call := newCallContext(nil, nil)
	if _, err := entry.fn(call, []Value{externValue, funcValue}); err != nil {
		t.Fatal(err)
	}
	call.expire()

	if got := escapedRef.Value(); got != "payload" {
		t.Fatalf("package-owned externref lost canonical lifetime: %v", got)
	}
	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("escaped %s must expire with its CallContext", name)
			}
		}()
		fn()
	}
	assertPanics("funcref", func() { _ = escapedFunc.Type() })
}

func TestHostFunctionOpaqueTokenLifecycle(t *testing.T) {
	standalone := MustWrapFunc(func() {})
	token := standalone.token
	if _, ok := hostFuncTokens.Load(token); !ok {
		t.Fatal("host function token was not registered")
	}
	if got := externRefValueFromToken(token, nil).ExternRef(); got != nil {
		t.Fatalf("host function token resolved as externref: %#v", got)
	}
	foreignModule := newModuleWithDataToken("foreign-host-token", token)
	if foreignModule == nil {
		t.Fatal("module allocation failed")
	}
	if got := foreignModule.HostData(); got != nil {
		t.Fatalf("host function token resolved as module data: %#v", got)
	}
	if err := foreignModule.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := hostFuncTokens.Load(token); !ok {
		t.Fatal("foreign module finalizer released a host function token")
	}
	if err := standalone.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := hostFuncTokens.Load(token); ok {
		t.Fatal("Function.Close did not release its host function token")
	}

	transferred := MustWrapFunc(func() {})
	transferredToken := transferred.token
	module := NewModule("host-token-owner")
	if err := module.AddFunction("f", transferred); err != nil {
		t.Fatal(err)
	}
	if err := transferred.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := hostFuncTokens.Load(transferredToken); !ok {
		t.Fatal("transferred wrapper released the module-owned callback token")
	}
	if err := module.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := hostFuncTokens.Load(transferredToken); ok {
		t.Fatal("Module.Close did not release an adopted callback token")
	}
}

func TestHostFunctionMayReturnCallScopedReference(t *testing.T) {
	target := MustWrapFunc(func() {})
	defer target.Close()
	call := newCallContext(nil, nil)
	borrowed := borrowedFunction(target.ptr, call)
	result := FuncRefValue(borrowed)

	entry := &hostFuncEntry{}
	if err := entry.retainResults(call, []Value{result}); err != nil {
		t.Fatalf("retain native call-scoped result: %v", err)
	}
	entry.close()
	call.expire()
}

func TestHostFunctionSelfResultDoesNotCreateLeaseCycle(t *testing.T) {
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()

	var self *Function
	self = MustWrapFunc(func() *Function { return self })
	defer self.Close()
	token := self.token

	out, err := exec.Invoke(self)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d results, want 1", len(out))
	}
	result := out[0].FuncRef()
	if result == nil || result.ptr != self.ptr {
		t.Fatal("host function did not return its own funcref")
	}
	if err := self.Close(); err != nil {
		t.Fatalf("self result created a lease cycle: %v", err)
	}
	if _, ok := hostFuncTokens.Load(token); ok {
		t.Fatal("closing the self-returning function did not release its host token")
	}
}

func TestHostFunctionExternalResultRetainsOwner(t *testing.T) {
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()

	target := MustWrapFunc(func() {})
	defer target.Close()
	producer := MustWrapFunc(func() *Function { return target })
	defer producer.Close()

	if _, err := exec.Invoke(producer); err != nil {
		t.Fatal(err)
	}
	if err := target.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("external funcref owner was not retained: %v", err)
	}
	if err := producer.Close(); err != nil {
		t.Fatalf("close producer: %v", err)
	}
	if err := target.Close(); err != nil {
		t.Fatalf("close released external funcref owner: %v", err)
	}
}

func TestHostFunctionModuleResultRoots(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		module := NewModule("same-module-result")
		target := MustWrapFunc(func() {})
		if err := module.AddFunction("target", target); err != nil {
			t.Fatal(err)
		}
		targetView, ok := module.Function("target")
		if !ok {
			t.Fatal("target function not found")
		}

		producer := MustWrapFunc(func() *Function { return targetView })
		if err := module.AddFunction("producer", producer); err != nil {
			t.Fatal(err)
		}
		producerView, ok := module.Function("producer")
		if !ok {
			t.Fatal("producer function not found")
		}
		exec, err := NewExecutor(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer exec.Close()
		out, err := exec.Invoke(producerView)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 1 || out[0].Kind() != ValKindFuncRef || out[0].IsNullRef() {
			t.Fatalf("unexpected producer result: %v", out)
		}
		if err := module.Close(); err != nil {
			t.Fatalf("same-module function result leased its own module: %v", err)
		}
	})

	t.Run("external module", func(t *testing.T) {
		external := NewModule("external-result-owner")
		target := MustWrapFunc(func() {})
		if err := external.AddFunction("target", target); err != nil {
			t.Fatal(err)
		}
		targetView, ok := external.Function("target")
		if !ok {
			t.Fatal("target function not found")
		}

		module := NewModule("external-result-producer")
		producer := MustWrapFunc(func() *Function { return targetView })
		if err := module.AddFunction("producer", producer); err != nil {
			t.Fatal(err)
		}
		producerView, ok := module.Function("producer")
		if !ok {
			t.Fatal("producer function not found")
		}
		exec, err := NewExecutor(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer exec.Close()
		if _, err := exec.Invoke(producerView); err != nil {
			t.Fatal(err)
		}
		if err := external.Close(); !errors.Is(err, ErrInUse) {
			t.Fatalf("external module was not retained: %v", err)
		}
		if err := module.Close(); err != nil {
			t.Fatalf("close producer module: %v", err)
		}
		if err := external.Close(); err != nil {
			t.Fatalf("external module lease was not released: %v", err)
		}
	})
}

func TestHostFunctionResultTypesValidated(t *testing.T) {
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()

	tests := []struct {
		name   string
		result ValType
		value  Value
	}{
		{name: "numeric kind", result: ValTypeI32(), value: I64(1)},
		{name: "reference kind", result: ValTypeFuncRef(), value: NullExternRef()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			function := mustNewHostFunction(t,
				FunctionType{Results: []ValType{tc.result}},
				func(*CallContext, []Value) ([]Value, error) {
					return []Value{tc.value}, nil
				},
			)
			defer function.Close()
			_, err := exec.Invoke(function)
			if err == nil {
				t.Fatal("mismatched host result type was accepted")
			}
			if !strings.Contains(err.Error(), "host function result 0 has kind") {
				t.Fatalf("unexpected result validation error: %v", err)
			}
		})
	}
}

func TestHostFunctionCopiesResultDescriptor(t *testing.T) {
	results := []ValType{ValTypeI32()}
	function := mustNewHostFunction(t,
		FunctionType{Results: results},
		func(*CallContext, []Value) ([]Value, error) {
			return []Value{I32(42)}, nil
		},
	)
	defer function.Close()
	results[0] = ValTypeI64()

	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()
	out, err := exec.Invoke(function)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].I32() != 42 {
		t.Fatalf("unexpected host result: %v", out)
	}
}

func TestHostFunctionPanicIsContained(t *testing.T) {
	boom := MustWrapFunc(func(a, b int32) int32 { panic("boom") })
	exec, inst := instantiateWithHost(t, boom)
	_, err := invokeCallHost(t, exec, inst, 1, 2)
	var panicErr *HostPanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("want *HostPanicError from contained panic, got %v", err)
	}
	if panicErr.Value != "boom" || len(panicErr.Stack) == 0 {
		t.Fatalf("contained panic lost value/stack: %+v", panicErr)
	}
	// The process surviving to this line is the real assertion.
}

func TestOrdinaryHostErrorPreservesIdentity(t *testing.T) {
	sentinel := errors.New("host backend unavailable")
	fail := MustWrapFunc(func(int32, int32) (int32, error) {
		return 0, fmt.Errorf("lookup: %w", sentinel)
	})
	exec, inst := instantiateWithHost(t, fail)
	_, err := invokeCallHost(t, exec, inst, 1, 2)
	if !errors.Is(err, sentinel) {
		t.Fatalf("host error identity lost through C: %v", err)
	}
	if err == nil || err.Error() != "lookup: host backend unavailable" {
		t.Fatalf("host error message lost through C: %v", err)
	}
}

func TestHostFunctionSuccessCodeErrorPreservesIdentity(t *testing.T) {
	custom := &Error{
		Category: ErrCategoryWASM,
		Code:     ErrCodeSuccess,
		Message:  "non-nil host error",
	}
	fail := MustWrapFunc(func(int32, int32) (int32, error) {
		return 0, custom
	})
	exec, inst := instantiateWithHost(t, fail)
	_, err := invokeCallHost(t, exec, inst, 1, 2)
	if err != custom {
		t.Fatalf("non-nil success-code host error lost identity: got %#v, want %#v", err, custom)
	}
}

func TestHostErrorMethodPanicIsContained(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		value string
	}{
		{name: "Is", err: panicIsHostError{}, value: "panic from error.Is"},
		{name: "As", err: panicAsHostError{}, value: "panic from error.As"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fail := MustWrapFunc(func(int32, int32) (int32, error) {
				return 0, tt.err
			})
			exec, inst := instantiateWithHost(t, fail)
			_, err := invokeCallHost(t, exec, inst, 1, 2)
			var panicErr *HostPanicError
			if !errors.As(err, &panicErr) {
				t.Fatalf("want contained *HostPanicError, got %v", err)
			}
			if panicErr.Value != tt.value || len(panicErr.Stack) == 0 {
				t.Fatalf("contained conversion panic lost value/stack: %+v", panicErr)
			}
		})
	}
}

func TestHostFunctionErrorRoundTrip(t *testing.T) {
	custom := &Error{Category: ErrCategoryUserLevel, Code: 0xBEEF}
	fail := mustNewHostFunction(t,
		FunctionType{
			Params:  []ValType{ValTypeI32(), ValTypeI32()},
			Results: []ValType{ValTypeI32()},
		},
		func(*CallContext, []Value) ([]Value, error) { return nil, custom })
	exec, inst := instantiateWithHost(t, fail)
	_, err := invokeCallHost(t, exec, inst, 1, 2)
	if !errors.Is(err, custom) {
		t.Fatalf("user error code lost through the engine: got %v", err)
	}
}

func TestHostFunctionTerminate(t *testing.T) {
	stop := mustNewHostFunction(t,
		FunctionType{
			Params:  []ValType{ValTypeI32(), ValTypeI32()},
			Results: []ValType{ValTypeI32()},
		},
		func(*CallContext, []Value) ([]Value, error) { return nil, Terminate })
	exec, inst := instantiateWithHost(t, stop)
	if _, err := invokeCallHost(t, exec, inst, 1, 2); err != nil {
		t.Fatalf("Terminate must surface as success, got %v", err)
	}
}

func TestCallContextMustNotEscape(t *testing.T) {
	var escaped *CallContext
	var escapedCopy CallContext
	var copied bool
	var escapedModule *Module
	var escapedExecutor *Executor
	var escapedCopyModule *Module
	var escapedCopyExecutor *Executor
	var escapedType FunctionType
	var escapedTypeSet bool
	leak := mustNewHostFunction(t,
		FunctionType{
			Params:  []ValType{ValTypeI32(), ValTypeI32()},
			Results: []ValType{ValTypeI32()},
		},
		func(call *CallContext, params []Value) ([]Value, error) {
			escaped = call
			escapedModule = call.Module()
			escapedExecutor = call.Executor()
			escapedCopy = *call
			copied = true
			escapedCopyModule = escapedCopy.Module()
			escapedCopyExecutor = escapedCopy.Executor()
			if f, ok := escapedModule.Function("call_host"); ok {
				escapedType = f.Type()
				escapedTypeSet = true
			}
			return []Value{I32(0)}, nil
		})
	exec, inst := instantiateWithHost(t, leak)
	if _, err := invokeCallHost(t, exec, inst, 1, 2); err != nil {
		t.Fatal(err)
	}
	if escapedModule == nil || escapedExecutor == nil ||
		!copied || escapedCopyModule == nil || escapedCopyExecutor == nil ||
		!escapedTypeSet {
		t.Fatal("host call did not populate escaped borrowed views")
	}

	assertPanics := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("using escaped %s must panic", name)
			}
		}()
		fn()
	}
	assertPanics("CallContext.Context", func() { _ = escaped.Context() })
	assertPanics("CallContext.Executor", func() { _ = escaped.Executor() })
	assertPanics("CallContext.Module", func() { _ = escaped.Module() })
	assertPanics("CallContext.Memory", func() { _ = escaped.Memory(0) })
	assertPanics("CallContext value copy Context", func() { _ = escapedCopy.Context() })
	assertPanics("CallContext value copy Executor", func() { _ = escapedCopy.Executor() })
	assertPanics("CallContext value copy Module", func() { _ = escapedCopy.Module() })
	assertPanics("CallContext value copy Memory", func() { _ = escapedCopy.Memory(0) })
	assertPanics("Module", func() { _ = escapedModule.Name() })
	assertPanics("Executor", func() { _, _ = escapedExecutor.Invoke(nil) })
	assertPanics("Module from CallContext value copy", func() { _ = escapedCopyModule.Name() })
	assertPanics("Executor from CallContext value copy", func() {
		_, _ = escapedCopyExecutor.Invoke(nil)
	})
	if len(escapedType.Results) != 1 {
		t.Fatal("copied FunctionType did not outlive its CallContext")
	}
}

func TestFunctionTransferSemantics(t *testing.T) {
	f := MustWrapFunc(func(a, b int32) int32 { return 0 })
	env := NewModule("env")
	defer env.Close()
	if err := env.AddFunction("host_add", f); err != nil {
		t.Fatal(err)
	}
	// After transfer the wrapper is inert: Close must not double-free.
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	// Adding a closed/transferred function is rejected in Go, not in C.
	if err := env.AddFunction("again", f); err == nil {
		t.Fatal("adding a transferred function must fail")
	}
}

func TestExternRefThroughHostFunction(t *testing.T) {
	type box struct{ s string }
	payload := &box{s: "hi"}
	ref := NewExternRef(payload)
	defer ref.Close()

	// The host function receives i32s from WASM here; externref transport
	// is exercised directly through values until a table fixture exists.
	val := ExternRefValue(ref)
	if got := val.ExternRef().Value().(*box); got != payload {
		t.Fatal("externref payload identity lost")
	}
	_ = fmt.Sprintf("%v", val) // String() must not panic on refs
}

func mustNewHostFunction(t *testing.T, typ FunctionType, fn HostFunc) *Function {
	t.Helper()
	function, err := NewFunction(typ, fn)
	if err != nil {
		t.Fatal(err)
	}
	return function
}

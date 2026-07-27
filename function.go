package wasmedge

// #include "shims.h"
import "C"

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// hostFuncTokens maps package-allocated C token addresses to Go callbacks.
// No Go pointer or numeric cgo.Handle value crosses the C boundary.
var hostFuncTokens sync.Map // map[unsafe.Pointer]*hostFuncEntry

// ErrReferenceCycle reports a host-function result that would create a
// cycle in the binding's native-resource lease graph. Rejecting the result
// keeps every involved function explicitly closable.
var ErrReferenceCycle = errors.New("wasmedge: reference ownership cycle")

// hostFuncRootGraphMu serializes dependency-cycle checks with root insertion
// and removal. Host callbacks attached to different executors may run
// concurrently even though each individual wrapper is single-operation.
var hostFuncRootGraphMu sync.Mutex

// HostFunc is the explicit host-function form: it receives the calling
// frame and the parameter values, and returns exactly as many values as the
// function type declares. Returning an error traps the calling WASM code;
// returning Terminate stops execution gracefully.
type HostFunc func(call *CallContext, params []Value) ([]Value, error)

type hostFuncEntry struct {
	fn       HostFunc
	function *C.WasmEdge_FunctionInstanceContext
	results  []ValType
	module   atomic.Uintptr
	mu       sync.Mutex
	roots    map[any]rootedReference
	// dependencies records canonical native host-function targets retained
	// by result roots. Provenance owners can be guest aliases, so graph
	// traversal must not infer the target from Value.owner.
	dependencies map[*C.WasmEdge_FunctionInstanceContext]struct{}
}

func (e *hostFuncEntry) validateResults(values []Value) error {
	if len(values) != len(e.results) {
		return fmt.Errorf("wasmedge: host function returned %d values, type wants %d",
			len(values), len(e.results))
	}
	for i, value := range values {
		got, want := value.Type(), e.results[i]
		if got.Kind() != want.Kind() {
			return fmt.Errorf(
				"wasmedge: host function result %d has kind %s, type wants %s",
				i, got.Kind(), want.Kind(),
			)
		}
		if got.Equal(want) {
			continue
		}
		// A concrete typed function reference is assignable to the classic
		// funcref top type. WasmEdge preserves that more precise dynamic type
		// when a guest returns ref.func, even through a funcref signature.
		if want.Equal(ValTypeFuncRef()) && got.Kind() == ValKindFuncRef {
			continue
		}
		return fmt.Errorf(
			"wasmedge: host function result %d has type %s, declared result is %s",
			i, got, want,
		)
	}
	return nil
}

func (e *hostFuncEntry) retainResults(call *CallContext, values []Value) error {
	hostFuncRootGraphMu.Lock()
	defer hostFuncRootGraphMu.Unlock()

	prepared := make([]rootedReference, 0, len(values))
	dependencies := make([]*C.WasmEdge_FunctionInstanceContext, 0, len(values))
	var callingModule uintptr
	if call != nil && call.ptr != nil {
		callingModule = uintptr(unsafe.Pointer(
			C.WasmEdge_CallingFrameGetModuleInstance(call.ptr),
		))
	}
	for _, value := range values {
		if !value.Type().IsRef() || value.owner == nil {
			continue
		}
		var target *C.WasmEdge_FunctionInstanceContext
		if value.Kind() == ValKindFuncRef {
			target = C.WasmEdge_ValueGetFuncRef(value.raw)
		}
		// A reference received from the current guest call is already owned
		// by the native instance. Returning it to that same call requires no
		// Go lease, and the CallContext intentionally cannot be retained past
		// the callback boundary.
		if callScopedOwner(value.owner) {
			continue
		}
		// The entry itself keeps result roots alive. Leasing this same
		// function would create an unbreakable cycle: Function.Close needs
		// the lease released, while releasing the lease requires Close to
		// finalize the entry. A wrapper for this native function therefore
		// needs no additional lease.
		if target == e.function {
			continue
		}
		// A guest module already owns references that came from that same
		// calling instance. Rooting the instance back through the imported
		// host module would create a module dependency cycle.
		if callingModule != 0 &&
			ownerChainContainsModule(value.owner, callingModule) {
			continue
		}
		// Once this host function is transferred to a module, the native
		// module already owns every function exported by that same module.
		// Leasing the module here would create a cycle: Module.Close cannot
		// finalize this entry until the entry releases the module lease.
		module := e.module.Load()
		if ownerChainContainsModule(value.owner, module) {
			continue
		}
		if targetEntry, ok := hostFuncEntryForFunction(target); ok &&
			module != 0 && targetEntry.module.Load() == module {
			continue
		}
		if ownerDependsOnModule(value.owner, module) {
			for i := range prepared {
				prepared[i].close()
			}
			return fmt.Errorf(
				"retain host result reference would reverse an import dependency: %w",
				ErrReferenceCycle,
			)
		}
		if target != nil && hostFunctionDependsOn(target, e.function, nil) {
			for i := range prepared {
				prepared[i].close()
			}
			return fmt.Errorf(
				"retain host result reference would create a dependency cycle: %w",
				ErrReferenceCycle,
			)
		}
		root, err := prepareReferenceRoot(value.owner)
		if err != nil {
			for i := range prepared {
				prepared[i].close()
			}
			return fmt.Errorf("retain host result reference: %w", err)
		}
		root.value = value
		root.hasValue = true
		prepared = append(prepared, root)
		if _, ok := hostFuncEntryForFunction(target); ok {
			dependencies = append(dependencies, target)
		}
	}
	e.mu.Lock()
	if e.roots == nil {
		e.roots = make(map[any]rootedReference)
	}
	if e.dependencies == nil {
		e.dependencies = make(map[*C.WasmEdge_FunctionInstanceContext]struct{})
	}
	for _, root := range prepared {
		if _, exists := e.roots[root.owner]; exists {
			root.close()
			continue
		}
		e.roots[root.owner] = root
	}
	for _, dependency := range dependencies {
		e.dependencies[dependency] = struct{}{}
	}
	e.mu.Unlock()
	return nil
}

func (e *hostFuncEntry) close() {
	hostFuncRootGraphMu.Lock()
	e.mu.Lock()
	roots := e.roots
	e.roots = nil
	e.dependencies = nil
	e.mu.Unlock()
	hostFuncRootGraphMu.Unlock()
	for _, root := range roots {
		root.close()
	}
}

// hostFunctionDependsOn reports whether start already has a transitive root
// edge to target. The global graph lock must be held by the caller.
func hostFunctionDependsOn(
	start *C.WasmEdge_FunctionInstanceContext,
	target *C.WasmEdge_FunctionInstanceContext,
	seen map[*hostFuncEntry]struct{},
) bool {
	if start == nil || target == nil {
		return false
	}
	if start == target {
		return true
	}
	entry, ok := hostFuncEntryForFunction(start)
	if !ok {
		return false
	}
	if seen == nil {
		seen = make(map[*hostFuncEntry]struct{})
	}
	if _, exists := seen[entry]; exists {
		return false
	}
	seen[entry] = struct{}{}

	entry.mu.Lock()
	dependencies := make(
		[]*C.WasmEdge_FunctionInstanceContext,
		0,
		len(entry.dependencies),
	)
	for dependency := range entry.dependencies {
		dependencies = append(dependencies, dependency)
	}
	entry.mu.Unlock()
	for _, dependency := range dependencies {
		if hostFunctionDependsOn(dependency, target, seen) {
			return true
		}
	}
	return false
}

func hostFuncEntryForFunction(
	function *C.WasmEdge_FunctionInstanceContext,
) (*hostFuncEntry, bool) {
	var found *hostFuncEntry
	hostFuncTokens.Range(func(_, raw any) bool {
		entry, ok := raw.(*hostFuncEntry)
		if ok && entry.function == function {
			found = entry
			return false
		}
		return true
	})
	return found, found != nil
}

func hostFuncEntryForToken(token unsafe.Pointer) (*hostFuncEntry, bool) {
	raw, ok := hostFuncTokens.Load(token)
	if !ok {
		return nil, false
	}
	entry, ok := raw.(*hostFuncEntry)
	return entry, ok
}

// unpublishHostFuncToken removes a callback from every Go-side graph before
// its native function address can be freed and reused. The opaque C token
// remains allocated until the caller finishes the native destructor.
func unpublishHostFuncToken(token unsafe.Pointer) bool {
	if token == nil {
		return false
	}
	raw, ok := hostFuncTokens.LoadAndDelete(token)
	if !ok {
		return false
	}
	if entry, ok := raw.(*hostFuncEntry); ok {
		entry.close()
	}
	return true
}

// Function is a function instance: either a host function created here, or
// a WASM function borrowed from a module or store.
type Function struct {
	ptr   *C.WasmEdge_FunctionInstanceContext
	token unsafe.Pointer // non-nil only for package-created host functions
	life  lifetime
}

// FunctionOption configures NewFunction.
type FunctionOption func(*functionOptions)

type functionOptions struct {
	cost uint64
}

// WithCost assigns the function's cost for statistics cost measuring.
func WithCost(cost uint64) FunctionOption {
	return func(o *functionOptions) { o.cost = cost }
}

// NewFunction creates a host function of the given type. The result is owned;
// adding it to a Module transfers ownership to the module. Invalid types,
// callbacks, or options match ErrInvalidArgument; native allocation failure
// matches ErrUnavailable.
func NewFunction(ft FunctionType, fn HostFunc, opts ...FunctionOption) (*Function, error) {
	if fn == nil {
		return nil, fmt.Errorf("host function is nil: %w", ErrInvalidArgument)
	}
	var o functionOptions
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("function option is nil: %w", ErrInvalidArgument)
		}
		opt(&o)
	}
	cft, freeType, err := ft.build()
	if err != nil {
		return nil, fmt.Errorf("function type: %w", err)
	}
	defer freeType()

	token := newOpaqueToken()
	if token == nil {
		return nil, fmt.Errorf("create host function token: %w", ErrUnavailable)
	}
	entry := &hostFuncEntry{
		fn:      fn,
		results: append([]ValType(nil), ft.Results...),
	}
	ptr := C.wasmedgego_functionCreate(cft, token, C.uint64_t(o.cost))
	if ptr == nil {
		deleteOpaqueToken(token)
		return nil, fmt.Errorf("create host function: %w", ErrUnavailable)
	}
	entry.function = ptr
	// Publish only after immutable native identity is fully initialized.
	// Concurrent graph scans can never observe a partially built entry.
	hostFuncTokens.Store(token, entry)
	f := &Function{ptr: ptr, token: token}
	arm(f, &f.life, "Function", func() {
		detached := unpublishHostFuncToken(token)
		C.WasmEdge_FunctionInstanceDelete(ptr)
		if detached {
			deleteOpaqueToken(token)
		}
	})
	return f, nil
}

func borrowedFunction(ptr *C.WasmEdge_FunctionInstanceContext, owner any) *Function {
	if ptr == nil {
		return nil
	}
	return &Function{ptr: ptr, life: borrowed(owner)}
}

func (f *Function) assertAlive() { f.life.assertAlive("Function") }

func (f *Function) acquireLease() (func(), error) {
	return f.life.acquire("Function")
}

func (f *Function) prepareInvocationReferences(
	values []Value,
) (func(bool), error) {
	if owner, ok := f.life.owner.(invocationReferenceRetainer); ok {
		return owner.prepareInvocationReferences(values)
	}
	return func(bool) {}, nil
}

// Type returns a copied descriptor of the function's type.
func (f *Function) Type() FunctionType {
	f.assertAlive()
	defer runtime.KeepAlive(f)
	ft, _ := functionTypeFromC(C.WasmEdge_FunctionInstanceGetFunctionType(f.ptr))
	return ft
}

// Close frees an owned host function that was NOT added to a module, and
// releases its Go closure. No-op for borrowed functions, after a transfer,
// and after the first call.
func (f *Function) Close() error {
	ptr, token := f.ptr, f.token
	return f.life.close(func() {
		detached := unpublishHostFuncToken(token)
		C.WasmEdge_FunctionInstanceDelete(ptr)
		if detached {
			deleteOpaqueToken(token)
		}
	})
}

// FuncRefValue returns a funcref WASM value referring to f.
func FuncRefValue(f *Function) Value {
	if f == nil {
		return NullFuncRef()
	}
	f.assertAlive()
	defer runtime.KeepAlive(f)
	return Value{raw: C.WasmEdge_ValueGenFuncRef(f.ptr), owner: f}
}

// FuncRef returns the function referred to by a funcref value (borrowed),
// or nil for a null funcref. It panics if the value is not a funcref.
func (v Value) FuncRef() *Function {
	v.mustKind(ValKindFuncRef)
	v.assertOwnerAlive()
	return borrowedFunction(C.WasmEdge_ValueGetFuncRef(v.raw), v.owner)
}

// WrapFunc derives a host function from a plain Go function using
// reflection: parameters and results map to WASM types, an optional leading
// *CallContext parameter receives the calling frame, and an optional
// trailing error result reports failures.
//
// Supported Go types: int32/uint32 (i32), int64/uint64 (i64), float32
// (f32), float64 (f64), [16]byte (v128), *ExternRef (externref), and
// *Function (funcref). A nil reference pointer maps to a null WASM
// reference.
//
//	add := wasmedge.MustWrapFunc(func(a, b int32) int32 { return a + b })
func WrapFunc(fn any) (*Function, error) {
	v := reflect.ValueOf(fn)
	if !v.IsValid() {
		return nil, fmt.Errorf("wasmedge: WrapFunc wants a func, got <nil>")
	}
	t := v.Type()
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("wasmedge: WrapFunc wants a func, got %s", t)
	}
	if v.IsNil() {
		return nil, fmt.Errorf("wasmedge: WrapFunc wants a non-nil func")
	}
	if t.IsVariadic() {
		return nil, fmt.Errorf("wasmedge: WrapFunc does not support variadic funcs")
	}

	callCtxType := reflect.TypeOf((*CallContext)(nil))
	errType := reflect.TypeOf((*error)(nil)).Elem()

	firstParam := 0
	takesCtx := t.NumIn() > 0 && t.In(0) == callCtxType
	if takesCtx {
		firstParam = 1
	}

	var params []ValType
	for i := firstParam; i < t.NumIn(); i++ {
		vt, err := goTypeToValType(t.In(i))
		if err != nil {
			return nil, fmt.Errorf("wasmedge: parameter %d: %w", i, err)
		}
		params = append(params, vt)
	}

	numOut := t.NumOut()
	returnsErr := numOut > 0 && t.Out(numOut-1) == errType
	if returnsErr {
		numOut--
	}
	var results []ValType
	for i := 0; i < numOut; i++ {
		vt, err := goTypeToValType(t.Out(i))
		if err != nil {
			return nil, fmt.Errorf("wasmedge: result %d: %w", i, err)
		}
		results = append(results, vt)
	}

	ft := FunctionType{Params: params, Results: results}

	adapter := func(call *CallContext, in []Value) ([]Value, error) {
		if len(in) != len(params) {
			return nil, fmt.Errorf("wasmedge: host function received %d parameters, type wants %d",
				len(in), len(params))
		}
		args := make([]reflect.Value, 0, t.NumIn())
		if takesCtx {
			args = append(args, reflect.ValueOf(call))
		}
		for i, val := range in {
			// Values delivered by the native callback have no Go-side owner.
			// Bound borrowed reference wrappers to the CallContext so they
			// cannot be used after the host call returns.
			if val.Type().IsRef() && val.owner == nil && call != nil {
				val.owner = call
			}
			gv, err := unpackGoValue(t.In(firstParam+i), val)
			if err != nil {
				return nil, fmt.Errorf("wasmedge: parameter %d: %w", firstParam+i, err)
			}
			args = append(args, gv)
		}
		outs := v.Call(args)
		if returnsErr {
			if errv := outs[len(outs)-1]; !errv.IsNil() {
				return nil, errv.Interface().(error)
			}
			outs = outs[:len(outs)-1]
		}
		ret := make([]Value, len(outs))
		for i, out := range outs {
			ret[i] = packGoValue(out)
		}
		return ret, nil
	}
	return NewFunction(ft, adapter)
}

// MustWrapFunc is WrapFunc that panics on error, for package-level and
// test-time registration.
func MustWrapFunc(fn any) *Function {
	f, err := WrapFunc(fn)
	if err != nil {
		panic(err)
	}
	return f
}

func goTypeToValType(t reflect.Type) (ValType, error) {
	switch t {
	case reflect.TypeOf([16]byte{}):
		return ValTypeV128(), nil
	case reflect.TypeOf((*ExternRef)(nil)):
		return ValTypeExternRef(), nil
	case reflect.TypeOf((*Function)(nil)):
		return ValTypeFuncRef(), nil
	}

	switch t.Kind() {
	case reflect.Int32, reflect.Uint32:
		return ValTypeI32(), nil
	case reflect.Int64, reflect.Uint64:
		return ValTypeI64(), nil
	case reflect.Float32:
		return ValTypeF32(), nil
	case reflect.Float64:
		return ValTypeF64(), nil
	default:
		return ValType{}, fmt.Errorf("unsupported Go type %s", t)
	}
}

func unpackGoValue(t reflect.Type, v Value) (reflect.Value, error) {
	kind := v.Kind()
	switch t.Kind() {
	case reflect.Int32:
		if err := requireValueKind(t, kind, ValKindI32); err != nil {
			return reflect.Value{}, err
		}
		return convertGoValue(t, v.I32()), nil
	case reflect.Uint32:
		if err := requireValueKind(t, kind, ValKindI32); err != nil {
			return reflect.Value{}, err
		}
		return convertGoValue(t, uint32(v.I32())), nil
	case reflect.Int64:
		if err := requireValueKind(t, kind, ValKindI64); err != nil {
			return reflect.Value{}, err
		}
		return convertGoValue(t, v.I64()), nil
	case reflect.Uint64:
		if err := requireValueKind(t, kind, ValKindI64); err != nil {
			return reflect.Value{}, err
		}
		return convertGoValue(t, uint64(v.I64())), nil
	case reflect.Float32:
		if err := requireValueKind(t, kind, ValKindF32); err != nil {
			return reflect.Value{}, err
		}
		return convertGoValue(t, v.F32()), nil
	case reflect.Float64:
		if err := requireValueKind(t, kind, ValKindF64); err != nil {
			return reflect.Value{}, err
		}
		return convertGoValue(t, v.F64()), nil
	case reflect.Array:
		if t != reflect.TypeOf([16]byte{}) {
			break
		}
		if err := requireValueKind(t, kind, ValKindV128); err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v.V128()), nil
	case reflect.Pointer:
		switch t {
		case reflect.TypeOf((*ExternRef)(nil)):
			if err := requireValueKind(t, kind, ValKindExternRef); err != nil {
				return reflect.Value{}, err
			}
			if v.IsNullRef() {
				return reflect.Zero(t), nil
			}
			ref := v.ExternRef()
			if ref == nil {
				return reflect.Value{}, fmt.Errorf(
					"wasmedge: cannot unpack foreign externref as %s", t)
			}
			return reflect.ValueOf(ref), nil
		case reflect.TypeOf((*Function)(nil)):
			if err := requireValueKind(t, kind, ValKindFuncRef); err != nil {
				return reflect.Value{}, err
			}
			if v.IsNullRef() {
				return reflect.Zero(t), nil
			}
			return reflect.ValueOf(v.FuncRef()), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("wasmedge: unsupported parameter type %s", t)
}

func requireValueKind(t reflect.Type, got, want ValKind) error {
	if got == want {
		return nil
	}
	return fmt.Errorf("cannot unpack %s as %s (want %s)", got, t, want)
}

func convertGoValue(t reflect.Type, v any) reflect.Value {
	rv := reflect.ValueOf(v)
	if rv.Type() != t {
		rv = rv.Convert(t)
	}
	return rv
}

func packGoValue(v reflect.Value) Value {
	switch v.Type() {
	case reflect.TypeOf([16]byte{}):
		return V128(v.Interface().([16]byte))
	case reflect.TypeOf((*ExternRef)(nil)):
		if v.IsNil() {
			return NullExternRef()
		}
		return ExternRefValue(v.Interface().(*ExternRef))
	case reflect.TypeOf((*Function)(nil)):
		if v.IsNil() {
			return NullFuncRef()
		}
		return FuncRefValue(v.Interface().(*Function))
	}

	switch v.Kind() {
	case reflect.Int32:
		return I32(int32(v.Int()))
	case reflect.Uint32:
		return I32(int32(uint32(v.Uint())))
	case reflect.Int64:
		return I64(v.Int())
	case reflect.Uint64:
		return I64(int64(v.Uint()))
	case reflect.Float32:
		return F32(float32(v.Float()))
	case reflect.Float64:
		return F64(v.Float())
	default:
		// WrapFunc validated the signature; reaching this is a bug.
		panic(fmt.Sprintf("wasmedge: unreachable result type %s", v.Type()))
	}
}

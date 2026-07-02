package wasmedge

// #include "shims.h"
import "C"

import (
	"fmt"
	"reflect"
	"runtime"
	"runtime/cgo"
)

// HostFunc is the explicit host-function form: it receives the calling
// frame and the parameter values, and returns exactly as many values as the
// function type declares. Returning an error traps the calling WASM code;
// returning Terminate stops execution gracefully.
type HostFunc func(call *CallContext, params []Value) ([]Value, error)

// hostFuncEntry is what a Function's cgo.Handle pins.
type hostFuncEntry struct {
	fn HostFunc
}

// Function is a function instance: either a host function created here, or
// a WASM function borrowed from a module or store.
type Function struct {
	ptr    *C.WasmEdge_FunctionInstanceContext
	handle cgo.Handle // non-zero only for owned host functions
	life   lifetime
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

// NewFunction creates a host function of the given type. The FunctionType
// remains caller-owned (the engine copies it). The result is owned; adding
// it to a Module transfers ownership to the module.
func NewFunction(ft *FunctionType, fn HostFunc, opts ...FunctionOption) *Function {
	if ft == nil || fn == nil {
		return nil
	}
	var o functionOptions
	for _, opt := range opts {
		opt(&o)
	}
	h := cgo.NewHandle(&hostFuncEntry{fn: fn})
	ptr := C.wasmedgego_functionCreate(ft.ptr, C.uintptr_t(h), C.uint64_t(o.cost))
	runtime.KeepAlive(ft)
	if ptr == nil {
		h.Delete()
		return nil
	}
	f := &Function{ptr: ptr, handle: h}
	arm(f, &f.life, "Function", func() {
		C.WasmEdge_FunctionInstanceDelete(ptr)
		h.Delete()
	})
	return f
}

func borrowedFunction(ptr *C.WasmEdge_FunctionInstanceContext, owner any) *Function {
	if ptr == nil {
		return nil
	}
	return &Function{ptr: ptr, life: borrowed(owner)}
}

// Type returns the function's type (borrowed).
func (f *Function) Type() *FunctionType {
	defer runtime.KeepAlive(f)
	return borrowedFunctionType(C.WasmEdge_FunctionInstanceGetFunctionType(f.ptr), f)
}

// Close frees an owned host function that was NOT added to a module, and
// releases its Go closure. No-op for borrowed functions, after a transfer,
// and after the first call.
func (f *Function) Close() error {
	ptr, h := f.ptr, f.handle
	return f.life.close(func() {
		C.WasmEdge_FunctionInstanceDelete(ptr)
		if h != 0 {
			h.Delete()
		}
	})
}

// FuncRefValue returns a funcref WASM value referring to f.
func FuncRefValue(f *Function) Value {
	if f == nil {
		return NullFuncRef()
	}
	defer runtime.KeepAlive(f)
	return Value{raw: C.WasmEdge_ValueGenFuncRef(f.ptr)}
}

// FuncRef returns the function referred to by a funcref value (borrowed),
// or nil for a null funcref. It panics if the value is not a funcref.
func (v Value) FuncRef() *Function {
	v.mustKind(ValKindFuncRef)
	return borrowedFunction(C.WasmEdge_ValueGetFuncRef(v.raw), nil)
}

// WrapFunc derives a host function from a plain Go function using
// reflection: parameters and results map to WASM types, an optional leading
// *CallContext parameter receives the calling frame, and an optional
// trailing error result reports failures.
//
// Supported Go types: int32/uint32 (i32), int64/uint64 (i64), float32
// (f32), float64 (f64).
//
//	add := wasmedge.MustWrapFunc(func(a, b int32) int32 { return a + b })
//
// TODO(intern-medium): A8 — extend the kind mapping with [16]byte (v128),
// *ExternRef (externref) and *Function (funcref) parameters and results.
// Add the ValType in goTypeToValType, the packing in packGoValue and the
// unpacking in unpackGoValue below; each switch is marked. Extend
// TestWrapFuncKinds in hostfunc_test.go with the new kinds.
func WrapFunc(fn any) (*Function, error) {
	v := reflect.ValueOf(fn)
	t := v.Type()
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("wasmedge: WrapFunc wants a func, got %s", t)
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

	ft := NewFunctionType(params, results)
	if ft == nil {
		return nil, fmt.Errorf("wasmedge: building function type failed")
	}
	defer ft.Close()

	adapter := func(call *CallContext, in []Value) ([]Value, error) {
		args := make([]reflect.Value, 0, t.NumIn())
		if takesCtx {
			args = append(args, reflect.ValueOf(call))
		}
		for i, val := range in {
			gv, err := unpackGoValue(t.In(firstParam+i), val)
			if err != nil {
				return nil, err
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
	f := NewFunction(ft, adapter)
	if f == nil {
		return nil, fmt.Errorf("wasmedge: creating function instance failed")
	}
	return f, nil
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
	switch t.Kind() {
	case reflect.Int32, reflect.Uint32:
		return ValTypeI32(), nil
	case reflect.Int64, reflect.Uint64:
		return ValTypeI64(), nil
	case reflect.Float32:
		return ValTypeF32(), nil
	case reflect.Float64:
		return ValTypeF64(), nil
	// TODO(intern-medium): A8 — add cases for [16]byte, *ExternRef and
	// *Function here.
	default:
		return ValType{}, fmt.Errorf("unsupported Go type %s", t)
	}
}

func unpackGoValue(t reflect.Type, v Value) (reflect.Value, error) {
	switch t.Kind() {
	case reflect.Int32:
		return reflect.ValueOf(v.I32()), nil
	case reflect.Uint32:
		return reflect.ValueOf(uint32(v.I32())), nil
	case reflect.Int64:
		return reflect.ValueOf(v.I64()), nil
	case reflect.Uint64:
		return reflect.ValueOf(uint64(v.I64())), nil
	case reflect.Float32:
		return reflect.ValueOf(v.F32()), nil
	case reflect.Float64:
		return reflect.ValueOf(v.F64()), nil
	// TODO(intern-medium): A8 — add cases for [16]byte, *ExternRef and
	// *Function here.
	default:
		return reflect.Value{}, fmt.Errorf("wasmedge: unsupported parameter type %s", t)
	}
}

func packGoValue(v reflect.Value) Value {
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
	// TODO(intern-medium): A8 — add cases for [16]byte, *ExternRef and
	// *Function here.
	default:
		// WrapFunc validated the signature; reaching this is a bug.
		panic(fmt.Sprintf("wasmedge: unreachable result kind %s", v.Kind()))
	}
}

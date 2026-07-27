package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Global is a global-variable instance.
type Global struct {
	ptr   *C.WasmEdge_GlobalInstanceContext
	life  lifetime
	roots *referenceRoots
}

// NewGlobal creates a standalone global for exporting from a host module.
// init must match the global type's value type. Descriptor or initializer
// validation failures match ErrInvalidArgument; native allocation failure
// matches ErrUnavailable.
func NewGlobal(gt GlobalType, init Value) (*Global, error) {
	if err := globalValueOwnerError(init); err != nil {
		return nil, err
	}
	if err := validateContainerValue("global initializer", gt.Value, init); err != nil {
		return nil, err
	}
	root, err := prepareReferenceRoot(init.owner)
	if err != nil {
		return nil, fmt.Errorf("global initializer reference: %w", err)
	}
	cgt, freeType, err := gt.build()
	if err != nil {
		root.close()
		return nil, err
	}
	defer freeType()
	ptr := C.WasmEdge_GlobalInstanceCreate(cgt, init.raw)
	runtime.KeepAlive(init)
	if ptr == nil {
		root.close()
		return nil, fmt.Errorf("create global: %w", ErrUnavailable)
	}
	g := &Global{ptr: ptr, roots: &referenceRoots{}}
	root.value = init
	root.hasValue = true
	g.roots.replace(0, root)
	arm(g, &g.life, "Global", func() { C.WasmEdge_GlobalInstanceDelete(ptr) })
	return g, nil
}

func borrowedGlobal(ptr *C.WasmEdge_GlobalInstanceContext, owner any) *Global {
	if ptr == nil {
		return nil
	}
	roots, _ := referenceState(owner, referenceObject{
		kind: referenceObjectGlobal,
		ptr:  uintptr(unsafe.Pointer(ptr)),
	})
	return &Global{ptr: ptr, life: borrowed(owner), roots: roots}
}

func (g *Global) assertAlive() { g.life.assertAlive("Global") }

func (g *Global) acquireLease() (func(), error) {
	return g.life.acquire("Global")
}

// Type returns a copied descriptor of the global's type.
func (g *Global) Type() GlobalType {
	g.assertAlive()
	defer runtime.KeepAlive(g)
	gt, _ := globalTypeFromC(C.WasmEdge_GlobalInstanceGetGlobalType(g.ptr))
	return gt
}

// Value returns the global's current value.
//
// For reference values created by this package, the result retains the
// wrapper that owns the reference. This lets a previously read value remain
// valid even when a mutable global is subsequently assigned a different
// reference.
func (g *Global) Value() Value {
	g.assertAlive()
	defer runtime.KeepAlive(g)

	v := C.WasmEdge_GlobalInstanceGetValue(g.ptr)
	var owner any = g
	if g.roots != nil {
		if rooted := g.roots.owner(0, Value{raw: v}); rooted != nil {
			owner = rooted
		}
	}
	return unpackValuesOwned([]C.WasmEdge_Value{v}, owner)[0]
}

// SetValue replaces the value of a mutable global. It returns a *Error with
// ErrCodeSetValueToConst for an immutable global. A value whose type does not
// exactly match the global's declared type is rejected with
// ErrInvalidArgument before calling the native setter.
//
// On success the global retains the Go owner of a reference value for as
// long as that value remains stored.
func (g *Global) SetValue(v Value) error {
	g.assertAlive()
	assertGlobalValueOwnerAlive(v)
	if err := validateContainerValue("global value", g.Type().Value, v); err != nil {
		return err
	}
	if v.owner != nil && g.roots == nil {
		return fmt.Errorf(
			"wasmedge: set Go reference through ephemeral global view: %w",
			ErrOwnership,
		)
	}
	var root rootedReference
	var err error
	if g.roots != nil {
		root, err = g.roots.prepare(v.owner)
	} else {
		root, err = prepareReferenceRoot(v.owner)
	}
	if err != nil {
		return fmt.Errorf("wasmedge: set global reference: %w", err)
	}
	defer runtime.KeepAlive(g)
	defer runtime.KeepAlive(v)

	if err := newResult(C.WasmEdge_GlobalInstanceSetValue(g.ptr, v.raw)); err != nil {
		root.close()
		return err
	}
	if g.roots != nil {
		root.value = v
		root.hasValue = true
		g.roots.replace(0, root)
	} else {
		root.close()
	}
	return nil
}

func assertGlobalValueOwnerAlive(v Value) {
	if owner, ok := v.owner.(aliveGuard); ok {
		owner.assertAlive()
	}
}

func globalValueOwnerError(v Value) (err error) {
	owner, ok := v.owner.(aliveGuard)
	if !ok {
		return nil
	}
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("global initializer owner is closed: %w", ErrClosed)
		}
	}()
	owner.assertAlive()
	return nil
}

// Close destroys an owned global that was not added to a module. No-op for
// borrowed views, after a transfer, and after the first call.
func (g *Global) Close() error {
	ptr := g.ptr
	return g.life.close(func() {
		C.WasmEdge_GlobalInstanceDelete(ptr)
		if g.roots != nil {
			g.roots.close()
			g.roots = nil
		}
	})
}

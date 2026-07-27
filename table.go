package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Table is a table instance holding reference values (funcref/externref).
type Table struct {
	ptr   *C.WasmEdge_TableInstanceContext
	life  lifetime
	roots *referenceRoots
}

// NewTable creates a standalone table instance for exporting from a host
// module, initialized with null references. A non-nullable reference element
// requires NewTableWithInit. Invalid descriptors match ErrInvalidArgument;
// a concrete size outside the host index range or native allocation failure
// matches ErrUnavailable.
func NewTable(tt TableType) (*Table, error) {
	if tt.Element.IsRef() && !tt.Element.IsRefNull() {
		return nil, fmt.Errorf(
			"non-nullable table element %s requires NewTableWithInit: %w",
			tt.Element, ErrInvalidArgument,
		)
	}
	if err := tt.validate(); err != nil {
		return nil, err
	}
	if err := validateStandaloneTableSize(tt.Limits.Min); err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}
	ctt, freeType, err := tt.build()
	if err != nil {
		return nil, err
	}
	defer freeType()
	ptr := C.WasmEdge_TableInstanceCreate(ctt)
	if ptr == nil {
		return nil, fmt.Errorf("create table: %w", ErrUnavailable)
	}
	return ownedTable(ptr), nil
}

// NewTableWithInit creates a table instance with every slot set to init,
// which must match the table's element type. Descriptor or initializer
// validation failures match ErrInvalidArgument. A concrete size outside the
// host index range or native allocation failure matches ErrUnavailable.
func NewTableWithInit(tt TableType, init Value) (*Table, error) {
	if err := tableValueOwnerError(init); err != nil {
		return nil, err
	}
	if err := validateContainerValue("table initializer", tt.Element, init); err != nil {
		return nil, err
	}
	if err := tt.validate(); err != nil {
		return nil, err
	}
	if err := validateStandaloneTableSize(tt.Limits.Min); err != nil {
		return nil, fmt.Errorf("create initialized table: %w", err)
	}
	root, err := prepareReferenceRoot(init.owner)
	if err != nil {
		return nil, fmt.Errorf("table initializer reference: %w", err)
	}
	ctt, freeType, err := tt.build()
	if err != nil {
		root.close()
		return nil, err
	}
	defer freeType()
	ptr := C.WasmEdge_TableInstanceCreateWithInit(ctt, init.raw)
	runtime.KeepAlive(init.owner)
	if ptr == nil {
		root.close()
		return nil, fmt.Errorf("create initialized table: %w", ErrUnavailable)
	}
	t := ownedTable(ptr)
	root.value = init
	root.hasValue = true
	t.roots.replace(referenceDefaultSlot, root)
	return t, nil
}

func ownedTable(ptr *C.WasmEdge_TableInstanceContext) *Table {
	t := &Table{ptr: ptr, roots: &referenceRoots{}}
	arm(t, &t.life, "Table", func() { C.WasmEdge_TableInstanceDelete(ptr) })
	return t
}

func borrowedTable(ptr *C.WasmEdge_TableInstanceContext, owner any) *Table {
	if ptr == nil {
		return nil
	}
	roots, _ := referenceState(owner, referenceObject{
		kind: referenceObjectTable,
		ptr:  uintptr(unsafe.Pointer(ptr)),
	})
	return &Table{ptr: ptr, life: borrowed(owner), roots: roots}
}

func (t *Table) assertAlive() { t.life.assertAlive("Table") }

func (t *Table) acquireLease() (func(), error) {
	return t.life.acquire("Table")
}

// Type returns a copied descriptor of the table's type.
func (t *Table) Type() TableType {
	t.assertAlive()
	defer runtime.KeepAlive(t)
	tt, _ := tableTypeFromC(C.WasmEdge_TableInstanceGetTableType(t.ptr))
	return tt
}

// Get returns the reference value at idx, bounds-checked by the engine.
func (t *Table) Get(idx uint64) (Value, error) {
	t.assertAlive()
	defer runtime.KeepAlive(t)
	var v Value
	if err := newResult(C.WasmEdge_TableInstanceGetData(t.ptr, &v.raw, C.uint64_t(idx))); err != nil {
		return Value{}, err
	}
	var owner any = t
	if t.roots != nil {
		if rooted := t.roots.owner(idx, v); rooted != nil {
			owner = rooted
		}
	}
	return unpackValuesOwned([]C.WasmEdge_Value{v.raw}, owner)[0], nil
}

// Set stores a reference value at idx. The Go wrapper checks its exact value
// type before the engine bounds-checks idx. For Go-backed externref and host
// funcref values, the resource that created v must remain open while the
// engine can observe the table entry.
func (t *Table) Set(idx uint64, v Value) error {
	t.assertAlive()
	assertTableValueOwnerAlive(v)
	if err := validateContainerValue("table value", t.Type().Element, v); err != nil {
		return err
	}
	if v.owner != nil && t.roots == nil {
		return fmt.Errorf("wasmedge: set Go reference through ephemeral table view: %w", ErrOwnership)
	}
	var root rootedReference
	var err error
	if t.roots != nil {
		root, err = t.roots.prepare(v.owner)
	} else {
		root, err = prepareReferenceRoot(v.owner)
	}
	if err != nil {
		return fmt.Errorf("wasmedge: set table reference: %w", err)
	}
	defer runtime.KeepAlive(t)
	err = newResult(C.WasmEdge_TableInstanceSetData(t.ptr, v.raw, C.uint64_t(idx)))
	runtime.KeepAlive(v.owner)
	if err != nil {
		root.close()
		return err
	}
	if t.roots != nil {
		root.value = v
		root.hasValue = true
		t.roots.replace(idx, root)
	} else {
		root.close()
	}
	return nil
}

// Size returns the current number of elements in the table.
func (t *Table) Size() uint64 {
	t.assertAlive()
	defer runtime.KeepAlive(t)
	return uint64(C.WasmEdge_TableInstanceGetSize(t.ptr))
}

// Grow increases the table by delta elements. New elements use the table's
// default initialization value. The engine returns an error if the resulting
// size would exceed the table's declared maximum. Host-index or size overflow
// matches ErrUnavailable before native growth.
func (t *Table) Grow(delta uint64) error {
	t.assertAlive()
	defer runtime.KeepAlive(t)
	current := uint64(C.WasmEdge_TableInstanceGetSize(t.ptr))
	if delta > ^uint64(0)-current {
		return fmt.Errorf(
			"grow table from %d by %d elements overflows its size: %w",
			current, delta, ErrUnavailable,
		)
	}
	if err := validateStandaloneTableSize(current + delta); err != nil {
		return fmt.Errorf("grow table from %d by %d elements: %w", current, delta, err)
	}
	return newResult(C.WasmEdge_TableInstanceGrow(t.ptr, C.uint64_t(delta)))
}

func assertTableValueOwnerAlive(v Value) {
	if owner, ok := v.owner.(aliveGuard); ok {
		owner.assertAlive()
	}
}

func tableValueOwnerError(v Value) (err error) {
	owner, ok := v.owner.(aliveGuard)
	if !ok {
		return nil
	}
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("table initializer owner is closed: %w", ErrClosed)
		}
	}()
	owner.assertAlive()
	return nil
}

// validateContainerValue rejects a value before passing it to WasmEdge's
// table/global storage APIs. WasmEdge 0.17.1 only distinguishes function-like
// and extern-like references in those APIs, which is not sufficient for the
// GC proposal's indexed reference types. Exact type equality is conservative
// and safe without the defining module's subtype graph.
func validateContainerValue(role string, expected ValType, value Value) error {
	actual := value.Type()
	if !actual.Equal(expected) {
		return fmt.Errorf(
			"%s type %s does not match declared type %s: %w",
			role, actual, expected, ErrInvalidArgument,
		)
	}
	if expected.IsRef() && !expected.IsRefNull() && value.IsNullRef() {
		return fmt.Errorf(
			"%s is null for non-nullable type %s: %w",
			role, expected, ErrInvalidArgument,
		)
	}
	return nil
}

// Close destroys an owned table that was not added to a module. No-op for
// borrowed views, after a transfer, and after the first call.
func (t *Table) Close() error {
	ptr := t.ptr
	return t.life.close(func() {
		C.WasmEdge_TableInstanceDelete(ptr)
		if t.roots != nil {
			t.roots.close()
			t.roots = nil
		}
	})
}

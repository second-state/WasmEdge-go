package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Table is a table instance holding reference values (funcref/externref).
type Table struct {
	ptr  *C.WasmEdge_TableInstanceContext
	life lifetime
}

// NewTable creates a standalone table instance for exporting from a host
// module, initialized with null references. The TableType remains
// caller-owned.
func NewTable(tt *TableType) *Table {
	if tt == nil {
		return nil
	}
	ptr := C.WasmEdge_TableInstanceCreate(tt.ptr)
	runtime.KeepAlive(tt)
	if ptr == nil {
		return nil
	}
	return ownedTable(ptr)
}

// NewTableWithInit creates a table instance with every slot set to init,
// which must match the table's element type.
func NewTableWithInit(tt *TableType, init Value) *Table {
	if tt == nil {
		return nil
	}
	ptr := C.WasmEdge_TableInstanceCreateWithInit(tt.ptr, init.raw)
	runtime.KeepAlive(tt)
	if ptr == nil {
		return nil
	}
	return ownedTable(ptr)
}

func ownedTable(ptr *C.WasmEdge_TableInstanceContext) *Table {
	t := &Table{ptr: ptr}
	arm(t, &t.life, "Table", func() { C.WasmEdge_TableInstanceDelete(ptr) })
	return t
}

func borrowedTable(ptr *C.WasmEdge_TableInstanceContext) *Table {
	if ptr == nil {
		return nil
	}
	return &Table{ptr: ptr, life: borrowed()}
}

// Get returns the reference value at idx, bounds-checked by the engine.
func (t *Table) Get(idx uint64) (Value, error) {
	defer runtime.KeepAlive(t)
	var v Value
	if err := newResult(C.WasmEdge_TableInstanceGetData(t.ptr, &v.raw, C.uint64_t(idx))); err != nil {
		return Value{}, err
	}
	return v, nil
}

// Set stores a reference value at idx, bounds- and type-checked by the
// engine.
func (t *Table) Set(idx uint64, v Value) error {
	defer runtime.KeepAlive(t)
	return newResult(C.WasmEdge_TableInstanceSetData(t.ptr, v.raw, C.uint64_t(idx)))
}

// TODO(intern-easy): A6 — bind the remaining table APIs on this type:
//
//	Size() uint64            -> WasmEdge_TableInstanceGetSize
//	Grow(delta uint64) error -> WasmEdge_TableInstanceGrow
//	Type() *TableType        -> WasmEdge_TableInstanceGetTableType (borrowed)
//
// Pattern: memory.go's PageCount/GrowPages/Type (KeepAlive discipline!).
// Extend module_test.go with TestTable: create a funcref table {Min: 2},
// Set/Get a FuncRefValue round-trip, Grow by 1, assert Size is 3, and
// assert Get(99) fails with a *Error.

// Close destroys an owned table that was not added to a module. No-op for
// borrowed views, after a transfer, and after the first call.
func (t *Table) Close() error {
	ptr := t.ptr
	return t.life.close(func() { C.WasmEdge_TableInstanceDelete(ptr) })
}

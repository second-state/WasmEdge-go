package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Mutability of a global.
type Mutability uint32

const (
	MutabilityConst Mutability = C.WasmEdge_Mutability_Const
	MutabilityVar   Mutability = C.WasmEdge_Mutability_Var
)

func (m Mutability) String() string {
	switch m {
	case MutabilityConst:
		return "const"
	case MutabilityVar:
		return "var"
	default:
		return "unknown"
	}
}

// ExternalType classifies an import or export entry.
type ExternalType uint32

const (
	ExternalTypeFunction ExternalType = C.WasmEdge_ExternalType_Function
	ExternalTypeTable    ExternalType = C.WasmEdge_ExternalType_Table
	ExternalTypeMemory   ExternalType = C.WasmEdge_ExternalType_Memory
	ExternalTypeGlobal   ExternalType = C.WasmEdge_ExternalType_Global
	ExternalTypeTag      ExternalType = C.WasmEdge_ExternalType_Tag
)

func (t ExternalType) String() string {
	switch t {
	case ExternalTypeFunction:
		return "function"
	case ExternalTypeTable:
		return "table"
	case ExternalTypeMemory:
		return "memory"
	case ExternalTypeGlobal:
		return "global"
	case ExternalTypeTag:
		return "tag"
	default:
		return "unknown"
	}
}

// FunctionType describes parameter and result types of a function.
type FunctionType struct {
	ptr  *C.WasmEdge_FunctionTypeContext
	life lifetime
}

// NewFunctionType creates a function type. Both slices may be empty. The
// result is owned: Close it unless it is consumed by a documented
// ownership-transferring call (engine consumers copy function types, so in
// practice: always Close).
func NewFunctionType(params, results []ValType) *FunctionType {
	cp, cr := packValTypes(params), packValTypes(results)
	ptr := C.WasmEdge_FunctionTypeCreate(
		valTypesPtr(cp), C.uint32_t(len(cp)), valTypesPtr(cr), C.uint32_t(len(cr)))
	if ptr == nil {
		return nil
	}
	t := &FunctionType{ptr: ptr}
	arm(t, &t.life, "FunctionType", func() { C.WasmEdge_FunctionTypeDelete(ptr) })
	return t
}

func borrowedFunctionType(ptr *C.WasmEdge_FunctionTypeContext, owner any) *FunctionType {
	if ptr == nil {
		return nil
	}
	return &FunctionType{ptr: ptr, life: borrowed(owner)}
}

// Parameters returns the parameter types.
func (t *FunctionType) Parameters() []ValType {
	defer runtime.KeepAlive(t)
	n := C.WasmEdge_FunctionTypeGetParametersLength(t.ptr)
	if n == 0 {
		return nil
	}
	buf := make([]C.WasmEdge_ValType, n)
	got := C.WasmEdge_FunctionTypeGetParameters(t.ptr, &buf[0], n)
	return unpackValTypes(buf[:min(got, n)])
}

// Results returns the result types.
func (t *FunctionType) Results() []ValType {
	defer runtime.KeepAlive(t)
	n := C.WasmEdge_FunctionTypeGetReturnsLength(t.ptr)
	if n == 0 {
		return nil
	}
	buf := make([]C.WasmEdge_ValType, n)
	got := C.WasmEdge_FunctionTypeGetReturns(t.ptr, &buf[0], n)
	return unpackValTypes(buf[:min(got, n)])
}

// Close frees the type. No-op on borrowed views and after the first call.
func (t *FunctionType) Close() error {
	ptr := t.ptr
	return t.life.close(func() { C.WasmEdge_FunctionTypeDelete(ptr) })
}

// TableType describes a table's element type and size limits.
type TableType struct {
	ptr  *C.WasmEdge_TableTypeContext
	life lifetime
}

// NewTableType creates a table type. refType must be a reference type
// (funcref or externref); the C API returns nil otherwise, and so does this.
func NewTableType(refType ValType, limits Limits) *TableType {
	clim, free := limits.build()
	defer free()
	ptr := C.WasmEdge_TableTypeCreate(refType.raw, clim)
	if ptr == nil {
		return nil
	}
	t := &TableType{ptr: ptr}
	arm(t, &t.life, "TableType", func() { C.WasmEdge_TableTypeDelete(ptr) })
	return t
}

func borrowedTableType(ptr *C.WasmEdge_TableTypeContext, owner any) *TableType {
	if ptr == nil {
		return nil
	}
	return &TableType{ptr: ptr, life: borrowed(owner)}
}

// RefType returns the table's element type.
func (t *TableType) RefType() ValType {
	defer runtime.KeepAlive(t)
	return ValType{raw: C.WasmEdge_TableTypeGetRefType(t.ptr)}
}

// Limits returns the table's size limits.
func (t *TableType) Limits() Limits {
	defer runtime.KeepAlive(t)
	return limitsFromC(C.WasmEdge_TableTypeGetLimit(t.ptr))
}

// Close frees the type. No-op on borrowed views and after the first call.
func (t *TableType) Close() error {
	ptr := t.ptr
	return t.life.close(func() { C.WasmEdge_TableTypeDelete(ptr) })
}

// MemoryType describes a linear memory's page limits.
type MemoryType struct {
	ptr  *C.WasmEdge_MemoryTypeContext
	life lifetime
}

// NewMemoryType creates a memory type.
func NewMemoryType(limits Limits) *MemoryType {
	clim, free := limits.build()
	defer free()
	ptr := C.WasmEdge_MemoryTypeCreate(clim)
	if ptr == nil {
		return nil
	}
	t := &MemoryType{ptr: ptr}
	arm(t, &t.life, "MemoryType", func() { C.WasmEdge_MemoryTypeDelete(ptr) })
	return t
}

func borrowedMemoryType(ptr *C.WasmEdge_MemoryTypeContext, owner any) *MemoryType {
	if ptr == nil {
		return nil
	}
	return &MemoryType{ptr: ptr, life: borrowed(owner)}
}

// Limits returns the memory's page limits.
func (t *MemoryType) Limits() Limits {
	defer runtime.KeepAlive(t)
	return limitsFromC(C.WasmEdge_MemoryTypeGetLimit(t.ptr))
}

// Close frees the type. No-op on borrowed views and after the first call.
func (t *MemoryType) Close() error {
	ptr := t.ptr
	return t.life.close(func() { C.WasmEdge_MemoryTypeDelete(ptr) })
}

// GlobalType describes a global's value type and mutability.
type GlobalType struct {
	ptr  *C.WasmEdge_GlobalTypeContext
	life lifetime
}

// NewGlobalType creates a global type.
func NewGlobalType(valType ValType, mut Mutability) *GlobalType {
	ptr := C.WasmEdge_GlobalTypeCreate(valType.raw, C.enum_WasmEdge_Mutability(mut))
	if ptr == nil {
		return nil
	}
	t := &GlobalType{ptr: ptr}
	arm(t, &t.life, "GlobalType", func() { C.WasmEdge_GlobalTypeDelete(ptr) })
	return t
}

func borrowedGlobalType(ptr *C.WasmEdge_GlobalTypeContext, owner any) *GlobalType {
	if ptr == nil {
		return nil
	}
	return &GlobalType{ptr: ptr, life: borrowed(owner)}
}

// ValType returns the global's value type.
func (t *GlobalType) ValType() ValType {
	defer runtime.KeepAlive(t)
	return ValType{raw: C.WasmEdge_GlobalTypeGetValType(t.ptr)}
}

// Mutability returns whether the global is const or var.
func (t *GlobalType) Mutability() Mutability {
	defer runtime.KeepAlive(t)
	return Mutability(C.WasmEdge_GlobalTypeGetMutability(t.ptr))
}

// Close frees the type. No-op on borrowed views and after the first call.
func (t *GlobalType) Close() error {
	ptr := t.ptr
	return t.life.close(func() { C.WasmEdge_GlobalTypeDelete(ptr) })
}

// TagType describes an exception-handling tag. Tags cannot be created via
// the C API; TagType values are always borrowed from modules or instances.
type TagType struct {
	ptr   *C.WasmEdge_TagTypeContext
	owner any // pins whatever owns the underlying tag type
}

func borrowedTagType(ptr *C.WasmEdge_TagTypeContext, owner any) *TagType {
	if ptr == nil {
		return nil
	}
	return &TagType{ptr: ptr, owner: owner}
}

// FunctionType returns the tag's associated function type (borrowed).
func (t *TagType) FunctionType() *FunctionType {
	defer runtime.KeepAlive(t)
	return borrowedFunctionType(C.WasmEdge_TagTypeGetFunctionType(t.ptr), t)
}

func packValTypes(ts []ValType) []C.WasmEdge_ValType {
	if len(ts) == 0 {
		return nil
	}
	buf := make([]C.WasmEdge_ValType, len(ts))
	for i, t := range ts {
		buf[i] = t.raw
	}
	return buf
}

// valTypesPtr returns the C array head for a packed slice (nil for empty).
func valTypesPtr(cts []C.WasmEdge_ValType) *C.WasmEdge_ValType {
	if len(cts) == 0 {
		return nil
	}
	return &cts[0]
}

func unpackValTypes(cts []C.WasmEdge_ValType) []ValType {
	out := make([]ValType, len(cts))
	for i, ct := range cts {
		out[i] = ValType{raw: ct}
	}
	return out
}

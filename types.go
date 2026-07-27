package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "fmt"

// Mutability of a global.
type Mutability uint32

// Mutability values describe whether a global can be updated.
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

// ExternalType values identify the kind of an import or export.
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

// FunctionType describes the parameter and result types of a function.
// It is a plain Go value. Params and Results are copied when the descriptor
// crosses the native API boundary.
type FunctionType struct {
	Params  []ValType
	Results []ValType
}

func functionTypeFromC(ptr *C.WasmEdge_FunctionTypeContext) (FunctionType, bool) {
	if ptr == nil {
		return FunctionType{}, false
	}
	var out FunctionType
	if n := C.WasmEdge_FunctionTypeGetParametersLength(ptr); n != 0 {
		buf := make([]C.WasmEdge_ValType, n)
		got := C.WasmEdge_FunctionTypeGetParameters(ptr, &buf[0], n)
		out.Params = unpackValTypes(buf[:min(got, n)])
	}
	if n := C.WasmEdge_FunctionTypeGetReturnsLength(ptr); n != 0 {
		buf := make([]C.WasmEdge_ValType, n)
		got := C.WasmEdge_FunctionTypeGetReturns(ptr, &buf[0], n)
		out.Results = unpackValTypes(buf[:min(got, n)])
	}
	return out, true
}

func (t FunctionType) build() (*C.WasmEdge_FunctionTypeContext, func(), error) {
	if err := validateValTypes("parameter", t.Params); err != nil {
		return nil, nil, err
	}
	if err := validateValTypes("result", t.Results); err != nil {
		return nil, nil, err
	}
	paramCount, err := checkedUint32Count(
		"function parameter type", uint64(len(t.Params)),
	)
	if err != nil {
		return nil, nil, err
	}
	resultCount, err := checkedUint32Count(
		"function result type", uint64(len(t.Results)),
	)
	if err != nil {
		return nil, nil, err
	}
	params, results := packValTypes(t.Params), packValTypes(t.Results)
	ptr := C.WasmEdge_FunctionTypeCreate(
		valTypesPtr(params), C.uint32_t(paramCount),
		valTypesPtr(results), C.uint32_t(resultCount),
	)
	if ptr == nil {
		return nil, nil, fmt.Errorf("create native function type: %w", ErrUnavailable)
	}
	return ptr, func() { C.WasmEdge_FunctionTypeDelete(ptr) }, nil
}

// TableType describes a table's element type and size limits.
type TableType struct {
	Element ValType
	Limits  Limits
}

func tableTypeFromC(ptr *C.WasmEdge_TableTypeContext) (TableType, bool) {
	if ptr == nil {
		return TableType{}, false
	}
	return TableType{
		Element: ValType{raw: C.WasmEdge_TableTypeGetRefType(ptr)},
		Limits:  limitsFromC(C.WasmEdge_TableTypeGetLimit(ptr)),
	}, true
}

func (t TableType) validate() error {
	if !t.Element.IsRef() {
		return fmt.Errorf("table element must be a reference type: %w", ErrInvalidArgument)
	}
	if t.Limits.Shared {
		return fmt.Errorf("table limits cannot be shared: %w", ErrInvalidArgument)
	}
	if err := validateTableLimits(t.Limits); err != nil {
		return fmt.Errorf("table limits: %w", err)
	}
	return nil
}

func (t TableType) build() (*C.WasmEdge_TableTypeContext, func(), error) {
	if err := t.validate(); err != nil {
		return nil, nil, err
	}
	limits, freeLimits := t.Limits.build()
	if limits == nil {
		return nil, nil, fmt.Errorf("create native table limits: %w", ErrUnavailable)
	}
	ptr := C.WasmEdge_TableTypeCreate(t.Element.raw, limits)
	freeLimits()
	if ptr == nil {
		return nil, nil, fmt.Errorf("create native table type: %w", ErrUnavailable)
	}
	return ptr, func() { C.WasmEdge_TableTypeDelete(ptr) }, nil
}

// MemoryType describes a linear memory's page limits.
type MemoryType struct {
	Limits Limits
}

func memoryTypeFromC(ptr *C.WasmEdge_MemoryTypeContext) (MemoryType, bool) {
	if ptr == nil {
		return MemoryType{}, false
	}
	return MemoryType{Limits: limitsFromC(C.WasmEdge_MemoryTypeGetLimit(ptr))}, true
}

func (t MemoryType) validate() error {
	if err := validateMemoryLimits(t.Limits); err != nil {
		return fmt.Errorf("memory limits: %w", err)
	}
	return nil
}

func (t MemoryType) build() (*C.WasmEdge_MemoryTypeContext, func(), error) {
	if err := t.validate(); err != nil {
		return nil, nil, err
	}
	limits, freeLimits := t.Limits.build()
	if limits == nil {
		return nil, nil, fmt.Errorf("create native memory limits: %w", ErrUnavailable)
	}
	ptr := C.WasmEdge_MemoryTypeCreate(limits)
	freeLimits()
	if ptr == nil {
		return nil, nil, fmt.Errorf("create native memory type: %w", ErrUnavailable)
	}
	return ptr, func() { C.WasmEdge_MemoryTypeDelete(ptr) }, nil
}

// GlobalType describes a global's value type and mutability.
type GlobalType struct {
	Value      ValType
	Mutability Mutability
}

func globalTypeFromC(ptr *C.WasmEdge_GlobalTypeContext) (GlobalType, bool) {
	if ptr == nil {
		return GlobalType{}, false
	}
	return GlobalType{
		Value:      ValType{raw: C.WasmEdge_GlobalTypeGetValType(ptr)},
		Mutability: Mutability(C.WasmEdge_GlobalTypeGetMutability(ptr)),
	}, true
}

func (t GlobalType) build() (*C.WasmEdge_GlobalTypeContext, func(), error) {
	if t.Value.Kind() == ValKindUnknown {
		return nil, nil, fmt.Errorf("global value type is invalid: %w", ErrInvalidArgument)
	}
	if t.Mutability != MutabilityConst && t.Mutability != MutabilityVar {
		return nil, nil, fmt.Errorf("global mutability %d is invalid: %w",
			uint32(t.Mutability), ErrInvalidArgument)
	}
	ptr := C.WasmEdge_GlobalTypeCreate(t.Value.raw, C.enum_WasmEdge_Mutability(t.Mutability))
	if ptr == nil {
		return nil, nil, fmt.Errorf("create native global type: %w", ErrUnavailable)
	}
	return ptr, func() { C.WasmEdge_GlobalTypeDelete(ptr) }, nil
}

// TagType describes an exception-handling tag. WasmEdge 0.17.1 exposes
// only the tag's function signature.
type TagType struct {
	Signature FunctionType
}

func tagTypeFromC(ptr *C.WasmEdge_TagTypeContext) (TagType, bool) {
	if ptr == nil {
		return TagType{}, false
	}
	signature, ok := functionTypeFromC(C.WasmEdge_TagTypeGetFunctionType(ptr))
	if !ok {
		return TagType{}, false
	}
	return TagType{Signature: signature}, true
}

func validateValTypes(role string, types []ValType) error {
	for i, typ := range types {
		if typ.Kind() == ValKindUnknown {
			return fmt.Errorf("%s %d has an invalid value type: %w",
				role, i, ErrInvalidArgument)
		}
	}
	return nil
}

func validateLimits(limits Limits) error {
	if limits.HasMax && limits.Max < limits.Min {
		return fmt.Errorf("maximum %d is smaller than minimum %d: %w",
			limits.Max, limits.Min, ErrInvalidArgument)
	}
	if limits.Shared && !limits.HasMax {
		return fmt.Errorf("shared memory requires a maximum: %w", ErrInvalidArgument)
	}
	return nil
}

const (
	maxMemory32Pages   = uint64(1 << 16)
	maxMemory64Pages   = uint64(1 << 48)
	maxTable32Elements = uint64(1<<32 - 1)
)

func maxInt() int { return int(^uint(0) >> 1) }

func validateStandaloneTableSize(size uint64) error {
	if size > uint64(maxInt()) {
		return fmt.Errorf(
			"table size %d exceeds the host index limit %d: %w",
			size, maxInt(), ErrUnavailable,
		)
	}
	return nil
}

func validateMemoryLimits(limits Limits) error {
	if err := validateLimits(limits); err != nil {
		return err
	}
	maximum := maxMemory32Pages
	if limits.Is64 {
		maximum = maxMemory64Pages
	}
	if limits.Min > maximum {
		return fmt.Errorf(
			"minimum %d exceeds the %d-page memory limit: %w",
			limits.Min, maximum, ErrInvalidArgument,
		)
	}
	if limits.HasMax && limits.Max > maximum {
		return fmt.Errorf(
			"maximum %d exceeds the %d-page memory limit: %w",
			limits.Max, maximum, ErrInvalidArgument,
		)
	}
	return nil
}

func validateTableLimits(limits Limits) error {
	if err := validateLimits(limits); err != nil {
		return err
	}
	if limits.Is64 {
		return nil
	}
	if limits.Min > maxTable32Elements {
		return fmt.Errorf(
			"minimum %d exceeds the 32-bit table limit: %w",
			limits.Min, ErrInvalidArgument,
		)
	}
	if limits.HasMax && limits.Max > maxTable32Elements {
		return fmt.Errorf(
			"maximum %d exceeds the 32-bit table limit: %w",
			limits.Max, ErrInvalidArgument,
		)
	}
	return nil
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

package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// ASTModule is a loaded (and possibly validated) module that has not been
// instantiated: the output of a Loader and the input of a Validator,
// Executor or VM registration.
type ASTModule struct {
	ptr  *C.WasmEdge_ASTModuleContext
	life lifetime
}

func ownedASTModule(ptr *C.WasmEdge_ASTModuleContext) *ASTModule {
	m := &ASTModule{ptr: ptr}
	arm(m, &m.life, "ASTModule", func() { C.WasmEdge_ASTModuleDelete(ptr) })
	return m
}

// Imports returns the module's import entries as borrowed views; they stay
// valid until the ASTModule is closed.
func (m *ASTModule) Imports() []*ImportType {
	defer runtime.KeepAlive(m)
	n := C.WasmEdge_ASTModuleListImportsLength(m.ptr)
	if n == 0 {
		return nil
	}
	buf := make([]*C.WasmEdge_ImportTypeContext, n)
	got := C.WasmEdge_ASTModuleListImports(m.ptr, &buf[0], n)
	out := make([]*ImportType, 0, got)
	for _, p := range buf[:min(got, n)] {
		out = append(out, &ImportType{ptr: p, ast: m})
	}
	return out
}

// Exports returns the module's export entries as borrowed views; they stay
// valid until the ASTModule is closed.
func (m *ASTModule) Exports() []*ExportType {
	defer runtime.KeepAlive(m)
	n := C.WasmEdge_ASTModuleListExportsLength(m.ptr)
	if n == 0 {
		return nil
	}
	buf := make([]*C.WasmEdge_ExportTypeContext, n)
	got := C.WasmEdge_ASTModuleListExports(m.ptr, &buf[0], n)
	out := make([]*ExportType, 0, got)
	for _, p := range buf[:min(got, n)] {
		out = append(out, &ExportType{ptr: p, ast: m})
	}
	return out
}

// Close frees the module. No-op after the first call.
func (m *ASTModule) Close() error {
	ptr := m.ptr
	return m.life.close(func() { C.WasmEdge_ASTModuleDelete(ptr) })
}

// ImportType is one import entry of an ASTModule (borrowed view).
type ImportType struct {
	ptr *C.WasmEdge_ImportTypeContext
	ast *ASTModule
}

// ExternalType reports what kind of entity is imported.
func (t *ImportType) ExternalType() ExternalType {
	defer runtime.KeepAlive(t.ast)
	return ExternalType(C.WasmEdge_ImportTypeGetExternalType(t.ptr))
}

// ModuleName returns the import's module name ("env" in "env.add").
func (t *ImportType) ModuleName() string {
	defer runtime.KeepAlive(t.ast)
	return goString(C.WasmEdge_ImportTypeGetModuleName(t.ptr))
}

// Name returns the import's external name ("add" in "env.add").
func (t *ImportType) Name() string {
	defer runtime.KeepAlive(t.ast)
	return goString(C.WasmEdge_ImportTypeGetExternalName(t.ptr))
}

// FunctionType returns the imported function's type (borrowed), or nil when
// the entry does not import a function.
func (t *ImportType) FunctionType() *FunctionType {
	defer runtime.KeepAlive(t.ast)
	return borrowedFunctionType(C.WasmEdge_ImportTypeGetFunctionType(t.ast.ptr, t.ptr))
}

// TODO(intern-easy): A3 — bind the remaining typed accessors on ImportType
// and ExportType:
//
//	(*ImportType) TableType()  *TableType   -> WasmEdge_ImportTypeGetTableType
//	(*ImportType) MemoryType() *MemoryType  -> WasmEdge_ImportTypeGetMemoryType
//	(*ImportType) GlobalType() *GlobalType  -> WasmEdge_ImportTypeGetGlobalType
//	(*ImportType) TagType()    *TagType     -> WasmEdge_ImportTypeGetTagType
//	(*ExportType) TableType()  *TableType   -> WasmEdge_ExportTypeGetTableType
//	(*ExportType) MemoryType() *MemoryType  -> WasmEdge_ExportTypeGetMemoryType
//	(*ExportType) GlobalType() *GlobalType  -> WasmEdge_ExportTypeGetGlobalType
//	(*ExportType) TagType()    *TagType     -> WasmEdge_ExportTypeGetTagType
//
// All results are borrowed (use borrowedTableType and friends; nil C pointer
// => nil). Pattern: FunctionType directly above. Add TestImportExportTypes
// to types_test.go per the note there.

// ExportType is one export entry of an ASTModule (borrowed view).
type ExportType struct {
	ptr *C.WasmEdge_ExportTypeContext
	ast *ASTModule
}

// ExternalType reports what kind of entity is exported.
func (t *ExportType) ExternalType() ExternalType {
	defer runtime.KeepAlive(t.ast)
	return ExternalType(C.WasmEdge_ExportTypeGetExternalType(t.ptr))
}

// Name returns the export's external name.
func (t *ExportType) Name() string {
	defer runtime.KeepAlive(t.ast)
	return goString(C.WasmEdge_ExportTypeGetExternalName(t.ptr))
}

// FunctionType returns the exported function's type (borrowed), or nil when
// the entry does not export a function.
func (t *ExportType) FunctionType() *FunctionType {
	defer runtime.KeepAlive(t.ast)
	return borrowedFunctionType(C.WasmEdge_ExportTypeGetFunctionType(t.ast.ptr, t.ptr))
}

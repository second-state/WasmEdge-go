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

func (m *ASTModule) assertAlive() { m.life.assertAlive("ASTModule") }

func (m *ASTModule) acquireLease() (func(), error) {
	return m.life.acquire("ASTModule")
}

// Imports returns the module's import entries as borrowed views; they stay
// valid until the ASTModule is closed.
func (m *ASTModule) Imports() []*ImportType {
	m.assertAlive()
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
	m.assertAlive()
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

func (t *ImportType) assertAlive() { t.ast.assertAlive() }

// ExternalType reports what kind of entity is imported.
func (t *ImportType) ExternalType() ExternalType {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return ExternalType(C.WasmEdge_ImportTypeGetExternalType(t.ptr))
}

// ModuleName returns the import's module name ("env" in "env.add").
func (t *ImportType) ModuleName() string {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return goString(C.WasmEdge_ImportTypeGetModuleName(t.ptr))
}

// Name returns the import's external name ("add" in "env.add").
func (t *ImportType) Name() string {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return goString(C.WasmEdge_ImportTypeGetExternalName(t.ptr))
}

// FunctionType returns a copied function descriptor and reports whether this
// entry imports a function.
func (t *ImportType) FunctionType() (FunctionType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return functionTypeFromC(C.WasmEdge_ImportTypeGetFunctionType(t.ast.ptr, t.ptr))
}

// TableType returns a copied table descriptor and reports whether this entry
// imports a table.
func (t *ImportType) TableType() (TableType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return tableTypeFromC(C.WasmEdge_ImportTypeGetTableType(t.ast.ptr, t.ptr))
}

// MemoryType returns a copied memory descriptor and reports whether this
// entry imports a memory.
func (t *ImportType) MemoryType() (MemoryType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return memoryTypeFromC(C.WasmEdge_ImportTypeGetMemoryType(t.ast.ptr, t.ptr))
}

// GlobalType returns a copied global descriptor and reports whether this
// entry imports a global.
func (t *ImportType) GlobalType() (GlobalType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return globalTypeFromC(C.WasmEdge_ImportTypeGetGlobalType(t.ast.ptr, t.ptr))
}

// TagType returns a copied tag descriptor and reports whether this entry
// imports a tag.
func (t *ImportType) TagType() (TagType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return tagTypeFromC(C.WasmEdge_ImportTypeGetTagType(t.ast.ptr, t.ptr))
}

// ExportType is one export entry of an ASTModule (borrowed view).
type ExportType struct {
	ptr *C.WasmEdge_ExportTypeContext
	ast *ASTModule
}

func (t *ExportType) assertAlive() { t.ast.assertAlive() }

// ExternalType reports what kind of entity is exported.
func (t *ExportType) ExternalType() ExternalType {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return ExternalType(C.WasmEdge_ExportTypeGetExternalType(t.ptr))
}

// Name returns the export's external name.
func (t *ExportType) Name() string {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return goString(C.WasmEdge_ExportTypeGetExternalName(t.ptr))
}

// FunctionType returns a copied function descriptor and reports whether this
// entry exports a function.
func (t *ExportType) FunctionType() (FunctionType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return functionTypeFromC(C.WasmEdge_ExportTypeGetFunctionType(t.ast.ptr, t.ptr))
}

// TableType returns a copied table descriptor and reports whether this entry
// exports a table.
func (t *ExportType) TableType() (TableType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return tableTypeFromC(C.WasmEdge_ExportTypeGetTableType(t.ast.ptr, t.ptr))
}

// MemoryType returns a copied memory descriptor and reports whether this
// entry exports a memory.
func (t *ExportType) MemoryType() (MemoryType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return memoryTypeFromC(C.WasmEdge_ExportTypeGetMemoryType(t.ast.ptr, t.ptr))
}

// GlobalType returns a copied global descriptor and reports whether this
// entry exports a global.
func (t *ExportType) GlobalType() (GlobalType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return globalTypeFromC(C.WasmEdge_ExportTypeGetGlobalType(t.ast.ptr, t.ptr))
}

// TagType returns a copied tag descriptor and reports whether this entry
// exports a tag.
func (t *ExportType) TagType() (TagType, bool) {
	t.assertAlive()
	defer runtime.KeepAlive(t.ast)
	return tagTypeFromC(C.WasmEdge_ExportTypeGetTagType(t.ast.ptr, t.ptr))
}

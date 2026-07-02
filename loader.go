package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import (
	"runtime"
	"unsafe"
)

// Loader parses WASM binaries (plain, universal-AOT or shared-library) into
// ASTModules.
type Loader struct {
	ptr  *C.WasmEdge_LoaderContext
	life lifetime
}

// NewLoader creates a loader honoring cfg (nil for defaults).
func NewLoader(cfg *Config) (*Loader, error) {
	ccfg, free := cfg.build()
	defer free()
	ptr := C.WasmEdge_LoaderCreate(ccfg)
	if ptr == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError, Message: "loader creation failed"}
	}
	l := &Loader{ptr: ptr}
	arm(l, &l.life, "Loader", func() { C.WasmEdge_LoaderDelete(ptr) })
	return l, nil
}

// LoadFile parses the WASM binary at path. The caller owns the result and
// must Close it (registration and instantiation copy what they need).
func (l *Loader) LoadFile(path string) (*ASTModule, error) {
	defer runtime.KeepAlive(l)
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var mod *C.WasmEdge_ASTModuleContext
	if err := newResult(C.WasmEdge_LoaderParseFromFile(l.ptr, &mod, cpath)); err != nil {
		return nil, err
	}
	return ownedASTModule(mod), nil
}

// LoadBytes parses a WASM binary from memory without copying it. The caller
// owns the result and must Close it.
func (l *Loader) LoadBytes(b []byte) (*ASTModule, error) {
	defer runtime.KeepAlive(l)
	var mod *C.WasmEdge_ASTModuleContext
	err := newResult(C.WasmEdge_LoaderParseFromBytes(l.ptr, &mod, wrapBytes(b)))
	runtime.KeepAlive(b)
	if err != nil {
		return nil, err
	}
	return ownedASTModule(mod), nil
}

// Serialize encodes an ASTModule back into WASM binary form (new in the
// 0.17 C API).
func (l *Loader) Serialize(m *ASTModule) ([]byte, error) {
	defer runtime.KeepAlive(l)
	defer runtime.KeepAlive(m)
	var buf C.WasmEdge_Bytes
	if err := newResult(C.WasmEdge_LoaderSerializeASTModule(l.ptr, m.ptr, &buf)); err != nil {
		return nil, err
	}
	return goBytesAndFree(buf), nil
}

// Close frees the loader. No-op after the first call.
func (l *Loader) Close() error {
	ptr := l.ptr
	return l.life.close(func() { C.WasmEdge_LoaderDelete(ptr) })
}

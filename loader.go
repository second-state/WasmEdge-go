package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

// ErrSerializeUnsupported reports a platform/runtime combination on which
// WasmEdge cannot safely serialize an AST module. It wraps
// errors.ErrUnsupported.
var ErrSerializeUnsupported = fmt.Errorf(
	"wasmedge: AST module serialization is unsafe on this platform with WasmEdge 0.17.x: %w",
	errors.ErrUnsupported,
)

// Loader parses WASM binaries (plain, universal-AOT or shared-library) into
// ASTModules.
type Loader struct {
	ptr  *C.WasmEdge_LoaderContext
	life lifetime
}

// NewLoader creates a loader honoring cfg (nil for defaults).
func NewLoader(cfg *Config) (*Loader, error) {
	ccfg, free, err := cfg.build()
	if err != nil {
		return nil, fmt.Errorf("create loader configuration: %w", err)
	}
	defer free()
	ptr := C.WasmEdge_LoaderCreate(ccfg)
	if ptr == nil {
		return nil, fmt.Errorf("create loader: %w", ErrUnavailable)
	}
	l := &Loader{ptr: ptr}
	arm(l, &l.life, "Loader", func() { C.WasmEdge_LoaderDelete(ptr) })
	return l, nil
}

func borrowedLoader(ptr *C.WasmEdge_LoaderContext, owner any) *Loader {
	if ptr == nil {
		return nil
	}
	return &Loader{ptr: ptr, life: borrowed(owner)}
}

func (l *Loader) assertAlive() { l.life.assertAlive("Loader") }

// LoadFile parses the WASM binary at path. The caller owns the result and
// must Close it (registration and instantiation copy what they need).
func (l *Loader) LoadFile(path string) (*ASTModule, error) {
	l.assertAlive()
	if err := validateCString("WASM path", path); err != nil {
		return nil, err
	}
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
	l.assertAlive()
	defer runtime.KeepAlive(l)
	bytes, err := wrapBytes(b)
	if err != nil {
		return nil, err
	}
	var mod *C.WasmEdge_ASTModuleContext
	err = newResult(C.WasmEdge_LoaderParseFromBytes(l.ptr, &mod, bytes))
	runtime.KeepAlive(b)
	if err != nil {
		return nil, err
	}
	return ownedASTModule(mod), nil
}

// Serialize encodes an ASTModule back into WASM binary form (new in the
// 0.17 C API). On darwin/arm64 it returns ErrSerializeUnsupported before
// entering C because WasmEdge 0.17.1 can terminate the process in this API.
func (l *Loader) Serialize(m *ASTModule) ([]byte, error) {
	l.assertAlive()
	if m == nil {
		return nil, fmt.Errorf("serialize requires a non-nil AST module: %w", ErrInvalidArgument)
	}
	m.assertAlive()
	if err := loaderSerializePlatformError(); err != nil {
		return nil, err
	}
	defer runtime.KeepAlive(l)
	defer runtime.KeepAlive(m)
	var buf C.WasmEdge_Bytes
	if err := newResult(C.WasmEdge_LoaderSerializeASTModule(l.ptr, m.ptr, &buf)); err != nil {
		return nil, err
	}
	return goBytesAndFree(buf)
}

// Close frees the loader. No-op after the first call.
func (l *Loader) Close() error {
	ptr := l.ptr
	return l.life.close(func() { C.WasmEdge_LoaderDelete(ptr) })
}

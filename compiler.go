package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Compiler is the AOT compiler: it turns WASM binaries into native shared
// libraries or universal WASM (AOT sections embedded), per
// Config.Compiler.OutputFormat.
type Compiler struct {
	ptr  *C.WasmEdge_CompilerContext
	life lifetime
}

// NewCompiler creates an AOT compiler honoring cfg (nil for defaults).
// Creation returns an error matching ErrUnavailable when the library was
// built without the AOT backend.
func NewCompiler(cfg *Config) (*Compiler, error) {
	ccfg, free, err := cfg.build()
	if err != nil {
		return nil, fmt.Errorf("create compiler configuration: %w", err)
	}
	defer free()
	ptr := C.WasmEdge_CompilerCreate(ccfg)
	if ptr == nil {
		return nil, fmt.Errorf("create AOT compiler: %w", ErrUnavailable)
	}
	c := &Compiler{ptr: ptr}
	arm(c, &c.life, "Compiler", func() { C.WasmEdge_CompilerDelete(ptr) })
	return c, nil
}

func (c *Compiler) assertAlive() { c.life.assertAlive("Compiler") }

// CompileFile compiles the WASM binary at inPath into outPath.
func (c *Compiler) CompileFile(inPath, outPath string) error {
	c.assertAlive()
	if err := validateCString("input WASM path", inPath); err != nil {
		return err
	}
	if err := validateCString("output artifact path", outPath); err != nil {
		return err
	}
	defer runtime.KeepAlive(c)
	cin := C.CString(inPath)
	defer C.free(unsafe.Pointer(cin))
	cout := C.CString(outPath)
	defer C.free(unsafe.Pointer(cout))
	return newResult(C.WasmEdge_CompilerCompile(c.ptr, cin, cout))
}

// CompileBytes compiles a WASM binary from memory into outPath.
func (c *Compiler) CompileBytes(b []byte, outPath string) error {
	c.assertAlive()
	if err := validateCString("output artifact path", outPath); err != nil {
		return err
	}
	bytes, err := wrapBytes(b)
	if err != nil {
		return err
	}
	defer runtime.KeepAlive(c)
	cout := C.CString(outPath)
	defer C.free(unsafe.Pointer(cout))
	err = newResult(C.WasmEdge_CompilerCompileFromBytes(c.ptr, bytes, cout))
	runtime.KeepAlive(b)
	return err
}

// Close frees the compiler. No-op after the first call.
func (c *Compiler) Close() error {
	ptr := c.ptr
	return c.life.close(func() { C.WasmEdge_CompilerDelete(ptr) })
}

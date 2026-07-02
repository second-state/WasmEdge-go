// C shims bridging WasmEdge callback typedefs to cgo-exported Go functions.
//
// Files containing //export directives cannot define C functions in their
// preambles, and the callback typedefs use const-qualified parameters that
// cgo-generated prototypes do not carry, so the adapters live here as real C
// code compiled into the package.

#ifndef WASMEDGEGO_SHIMS_H
#define WASMEDGEGO_SHIMS_H

#include <stdint.h>
#include <wasmedge/wasmedge.h>

// Install (enable != 0) or clear (enable == 0) the process-wide engine log
// callback, routing WasmEdge log records to the exported Go handler.
void wasmedgego_setLogCallback(int enable);

// Create a host function instance whose invocations bounce through the
// exported Go trampoline. Handle is a runtime/cgo.Handle identifying the Go
// closure; it travels via the binding's `This` pointer.
WasmEdge_FunctionInstanceContext *
wasmedgego_functionCreate(const WasmEdge_FunctionTypeContext *Type,
                          uintptr_t Handle, uint64_t Cost);

// Create a named module instance carrying a cgo.Handle as host data; the
// engine calls the exported Go finalizer (which releases the handle) when
// the module instance is destroyed.
WasmEdge_ModuleInstanceContext *
wasmedgego_moduleCreateWithData(WasmEdge_String Name, uintptr_t Handle);

#endif // WASMEDGEGO_SHIMS_H

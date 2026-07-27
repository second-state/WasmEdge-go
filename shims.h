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

// Allocate and release a one-byte opaque token in C memory. Go keeps the
// token address in a package-private registry; no cgo.Handle value is ever
// reinterpreted as a C pointer.
void *wasmedgego_tokenCreate(void);
void wasmedgego_tokenDelete(void *Token);

// Open and close binding-owned null-device descriptors for a WASI stdio
// discard policy. On Windows these are CRT descriptors, which is the form
// WasmEdge's INode::fromFd expects rather than raw HANDLE values.
int wasmedgego_wasiOpenNullFds(int32_t *StdInFd, int32_t *StdOutFd,
                               int32_t *StdErrFd, uint64_t *StdInHandler,
                               uint64_t *StdOutHandler,
                               uint64_t *StdErrHandler);
void wasmedgego_wasiCloseFds(int32_t StdInFd, int32_t StdOutFd,
                             int32_t StdErrFd);
int wasmedgego_wasiHandlerIsOpen(uint64_t Handler);

// Marshal v128 values as explicit low/high lanes so Go's [16]byte API keeps
// little-endian lane order even on big-endian targets such as s390x.
WasmEdge_Value wasmedgego_valueGenV128(uint64_t Low, uint64_t High);
void wasmedgego_valueGetV128(WasmEdge_Value Value, uint64_t *Low,
                             uint64_t *High);

// Install (enable != 0) or clear (enable == 0) the process-wide engine log
// callback, routing WasmEdge log records to the exported Go handler.
void wasmedgego_setLogCallback(int enable);

// Create a host function instance whose invocations bounce through the
// exported Go trampoline. Token is a C-allocated opaque address identifying
// the Go callback in a package-private registry.
WasmEdge_FunctionInstanceContext *
wasmedgego_functionCreate(const WasmEdge_FunctionTypeContext *Type, void *Token,
                          uint64_t Cost);

// Create a named module instance carrying an opaque C token as host data; the
// engine calls the exported Go finalizer when the module instance is
// destroyed.
WasmEdge_ModuleInstanceContext *
wasmedgego_moduleCreateWithData(WasmEdge_String Name, void *Token);

#endif // WASMEDGEGO_SHIMS_H

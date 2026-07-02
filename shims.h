// C shims bridging WasmEdge callback typedefs to cgo-exported Go functions.
//
// Files containing //export directives cannot define C functions in their
// preambles, and the callback typedefs use const-qualified parameters that
// cgo-generated prototypes do not carry, so the adapters live here as real C
// code compiled into the package.

#ifndef WASMEDGEGO_SHIMS_H
#define WASMEDGEGO_SHIMS_H

#include <wasmedge/wasmedge.h>

// Install (enable != 0) or clear (enable == 0) the process-wide engine log
// callback, routing WasmEdge log records to the exported Go handler.
void wasmedgego_setLogCallback(int enable);

#endif // WASMEDGEGO_SHIMS_H

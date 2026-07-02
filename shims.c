#include "shims.h"
#include "_cgo_export.h"

static void wasmedgego_logShim(const WasmEdge_LogMessage *Msg) {
  // Cast away const: the cgo-generated prototype for the exported Go
  // function has no const qualifier. The Go side treats it as read-only.
  wasmedgego_logCallback((WasmEdge_LogMessage *)Msg);
}

void wasmedgego_setLogCallback(int enable) {
  WasmEdge_LogSetCallback(enable ? wasmedgego_logShim : NULL);
}

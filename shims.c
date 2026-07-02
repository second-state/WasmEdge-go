#include "shims.h"
#include "_cgo_export.h"

// ---- logging ---------------------------------------------------------------

static void wasmedgego_logShim(const WasmEdge_LogMessage *Msg) {
  // Cast away const: the cgo-generated prototype for the exported Go
  // function has no const qualifier. The Go side treats it as read-only.
  wasmedgego_logCallback((WasmEdge_LogMessage *)Msg);
}

void wasmedgego_setLogCallback(int enable) {
  WasmEdge_LogSetCallback(enable ? wasmedgego_logShim : NULL);
}

// ---- host functions --------------------------------------------------------

static WasmEdge_Result
wasmedgego_wrapShim(void *This, void *Data,
                    const WasmEdge_CallingFrameContext *CallFrame,
                    const WasmEdge_Value *Params, const uint32_t ParamLen,
                    WasmEdge_Value *Returns, const uint32_t ReturnLen) {
  (void)Data;
  return wasmedgego_hostFuncInvoke(
      (uintptr_t)This, (WasmEdge_CallingFrameContext *)CallFrame,
      (WasmEdge_Value *)Params, ParamLen, Returns, ReturnLen);
}

WasmEdge_FunctionInstanceContext *
wasmedgego_functionCreate(const WasmEdge_FunctionTypeContext *Type,
                          uintptr_t Handle, uint64_t Cost) {
  return WasmEdge_FunctionInstanceCreateBinding(Type, wasmedgego_wrapShim,
                                                (void *)Handle, NULL, Cost);
}

// ---- module host data ------------------------------------------------------

static void wasmedgego_moduleDataFinalizeShim(void *Data) {
  wasmedgego_moduleDataFinalize((uintptr_t)Data);
}

WasmEdge_ModuleInstanceContext *
wasmedgego_moduleCreateWithData(WasmEdge_String Name, uintptr_t Handle) {
  return WasmEdge_ModuleInstanceCreateWithData(
      Name, (void *)Handle, wasmedgego_moduleDataFinalizeShim);
}

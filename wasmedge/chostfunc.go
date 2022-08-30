package wasmedge

/*
#include <wasmedge/wasmedge.h>

// The gateway function
WasmEdge_Result
wasmedgego_HostFuncInvoke(void *Func, void *Data,
                          const WasmEdge_CallingFrameContext *CallFrameCxt,
                          const WasmEdge_Value *Params, const uint32_t ParamLen,
                          WasmEdge_Value *Returns, const uint32_t ReturnLen) {
  WasmEdge_Result wasmedgego_HostFuncInvokeImpl(
      void *Func, void *Data, const WasmEdge_CallingFrameContext *CallFrameCxt,
      const WasmEdge_Value *Params, const uint32_t ParamLen,
      WasmEdge_Value *Returns, const uint32_t ReturnLen);
  return wasmedgego_HostFuncInvokeImpl(Func, Data, CallFrameCxt, Params,
                                       ParamLen, Returns, ReturnLen);
}
*/
import "C"

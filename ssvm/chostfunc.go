package ssvm

/*
#include <ssvm.h>
// The gateway function
SSVM_Result ssvmgo_HostFuncInvoke(void *Func, void *Data,
                           SSVM_MemoryInstanceContext *MemCxt,
                           const SSVM_Value *Params, const uint32_t ParamLen,
                           SSVM_Value *Returns, const uint32_t ReturnLen)
{
    SSVM_Result ssvmgo_HostFuncInvokeImpl(
        void *Func, void *Data,
        SSVM_MemoryInstanceContext *MemCxt,
        const SSVM_Value *Params, const uint32_t ParamLen,
        SSVM_Value *Returns, const uint32_t ReturnLen);
    return ssvmgo_HostFuncInvokeImpl(Func, Data, MemCxt, Params, ParamLen, Returns, ReturnLen);
}
*/
import "C"

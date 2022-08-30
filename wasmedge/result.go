package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type Result struct {
	code uint32
}

var (
	Result_Success   = Result{code: 0}
	Result_Terminate = Result{code: 1}
	Result_Fail      = Result{code: 2}
)

type ErrCategory C.enum_WasmEdge_ErrCategory

const (
	ErrCategory_WASM      = ErrCategory(C.WasmEdge_ErrCategory_WASM)
	ErrCategory_UserLevel = ErrCategory(C.WasmEdge_ErrCategory_UserLevelError)
)

func newError(res C.WasmEdge_Result) *Result {
	if C.WasmEdge_ResultOK(res) {
		return nil
	}
	return &Result{code: uint32(res.Code)}
}

func NewResult(cate ErrCategory, code int) Result {
	res := C.WasmEdge_ResultGen(C.enum_WasmEdge_ErrCategory(cate), C.uint32_t(code))
	return Result{
		code: uint32(res.Code),
	}
}

func (res *Result) Error() string {
	return C.GoString(C.WasmEdge_ResultGetMessage(C.WasmEdge_Result{Code: C.uint32_t(res.code)}))
}

func (res *Result) GetCode() int {
	return int(C.WasmEdge_ResultGetCode(C.WasmEdge_Result{Code: C.uint32_t(res.code)}))
}

func (res *Result) GetErrorCategory() ErrCategory {
	return ErrCategory(C.WasmEdge_ResultGetCategory(C.WasmEdge_Result{Code: C.uint32_t(res.code)}))
}

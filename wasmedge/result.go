package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type Result struct {
	code uint8
}

var (
	Result_Success   = Result{code: 0}
	Result_Terminate = Result{code: 1}
	Result_Fail      = Result{code: 2}
)

func newError(res C.WasmEdge_Result) *Result {
	if C.WasmEdge_ResultOK(res) {
		return nil
	}
	return &Result{
		code: uint8(res.Code),
	}
}

func (res *Result) Error() string {
	return C.GoString(C.WasmEdge_ResultGetMessage(C.WasmEdge_Result{Code: C.uint8_t(res.code)}))
}

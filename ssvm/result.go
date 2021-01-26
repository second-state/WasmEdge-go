package ssvm

// #include <ssvm.h>
import "C"

type Result struct {
	msg string
}

func newError(res C.SSVM_Result) *Result {
	if C.SSVM_ResultOK(res) {
		return nil
	}
	return &Result{
		msg: C.GoString(C.SSVM_ResultGetMessage(res)),
	}
}

func (res *Result) Error() string {
	return res.msg
}

package ssvm

// #include <ssvm.h>
import "C"

type FunctionType struct {
	_params  []C.enum_SSVM_ValType
	_returns []C.enum_SSVM_ValType
}

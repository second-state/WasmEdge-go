package ssvm

// #include <ssvm.h>
import "C"

func toSSVMStringWrap(str string) C.SSVM_String {
	return C.SSVM_StringWrap(C._GoStringPtr(str), C.uint32_t(C._GoStringLen(str)))
}

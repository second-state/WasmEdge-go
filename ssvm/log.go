package ssvm

// #include <ssvm.h>
import "C"

func SetLogErrorLevel() {
	C.SSVM_LogSetErrorLevel()
}

func SetLogDebugLevel() {
	C.SSVM_LogSetDebugLevel()
}

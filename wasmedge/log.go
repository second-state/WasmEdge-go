package wasmedge

// #include <wasmedge.h>
import "C"

func SetLogErrorLevel() {
	C.WasmEdge_LogSetErrorLevel()
}

func SetLogDebugLevel() {
	C.WasmEdge_LogSetDebugLevel()
}

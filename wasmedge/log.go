package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

func SetLogErrorLevel() {
	C.WasmEdge_LogSetErrorLevel()
}

func SetLogDebugLevel() {
	C.WasmEdge_LogSetDebugLevel()
}

func SetLogOff() {
	C.WasmEdge_LogOff()
}

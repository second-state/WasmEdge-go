package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

func RunWasmEdgeAOTCompilerCLI(argv []string) int {
	cargv := toCStringArray(argv)
	var ptrcargv *(*C.char) = nil
	if len(cargv) > 0 {
		ptrcargv = &cargv[0]
	}
	ret := C.WasmEdge_Driver_Compiler(C.int(len(cargv)), ptrcargv)
	freeCStringArray(cargv)
	return int(ret)
}

func RunWasmEdgeCLI(argv []string) int {
	cargv := toCStringArray(argv)
	var ptrcargv *(*C.char) = nil
	if len(cargv) > 0 {
		ptrcargv = &cargv[0]
	}
	ret := C.WasmEdge_Driver_Tool(C.int(len(cargv)), ptrcargv)
	freeCStringArray(cargv)
	return int(ret)
}

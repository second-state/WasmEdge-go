package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type LogLevel C.WasmEdge_LogLevel

const (
	LogLevel_Trace    = LogLevel(C.WasmEdge_LogLevel_Trace)
	LogLevel_Debug    = LogLevel(C.WasmEdge_LogLevel_Debug)
	LogLevel_Info     = LogLevel(C.WasmEdge_LogLevel_Info)
	LogLevel_Warn     = LogLevel(C.WasmEdge_LogLevel_Warn)
	LogLevel_Error    = LogLevel(C.WasmEdge_LogLevel_Error)
	LogLevel_Critical = LogLevel(C.WasmEdge_LogLevel_Critical)
)

func SetLogLevel(level LogLevel) {
	C.WasmEdge_LogSetLevel(C.WasmEdge_LogLevel(level))
}

func SetLogErrorLevel() {
	C.WasmEdge_LogSetErrorLevel()
}

func SetLogDebugLevel() {
	C.WasmEdge_LogSetDebugLevel()
}

func SetLogOff() {
	C.WasmEdge_LogOff()
}

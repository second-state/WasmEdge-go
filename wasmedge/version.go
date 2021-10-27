package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

func GetVersion() string {
	return C.GoString(C.WasmEdge_VersionGet())
}

func GetVersionMajor() uint {
	return uint(C.WasmEdge_VersionGetMajor())
}

func GetVersionMinor() uint {
	return uint(C.WasmEdge_VersionGetMinor())
}

func GetVersionPatch() uint {
	return uint(C.WasmEdge_VersionGetPatch())
}

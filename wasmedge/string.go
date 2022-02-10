package wasmedge

// #include <wasmedge/wasmedge.h>
// #include <stdlib.h>
import "C"
import "unsafe"

func toWasmEdgeStringWrap(str string) C.WasmEdge_String {
	return C.WasmEdge_StringWrap(C._GoStringPtr(str), C.uint32_t(C._GoStringLen(str)))
}

func fromWasmEdgeString(str C.WasmEdge_String) string {
	if int(str.Length) > 0 {
		return C.GoStringN(str.Buf, C.int32_t(str.Length))
	}
	return ""
}

func toCStringArray(strs []string) []*C.char {
	cstrs := make([]*C.char, len(strs))
	for i, str := range strs {
		cstrs[i] = C.CString(str)
	}
	return cstrs
}

func freeCStringArray(cstrs []*C.char) {
	for _, cstr := range cstrs {
		C.free(unsafe.Pointer(cstr))
	}
}

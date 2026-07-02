package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import (
	"runtime"
	"unsafe"
)

// newWEString copies s into a C-owned WasmEdge_String. The caller must free
// it with freeWEString, typically via defer at the call site:
//
//	name := newWEString("fib")
//	defer freeWEString(name)
//
// A copying helper is used instead of WasmEdge_StringWrap so call sites need
// no runtime.KeepAlive discipline; names are short and infrequent, so the
// copy is irrelevant. Large payloads (wasm binaries) use wrapBytes instead.
func newWEString(s string) C.WasmEdge_String {
	if len(s) == 0 {
		return C.WasmEdge_String{}
	}
	return C.WasmEdge_StringCreateByBuffer(
		(*C.char)(unsafe.Pointer(unsafe.StringData(s))), C.uint32_t(len(s)))
}

func freeWEString(s C.WasmEdge_String) {
	C.WasmEdge_StringDelete(s)
}

// goString copies a WasmEdge_String view into a Go string. Safe on borrowed
// views (names returned by List/Get APIs) because the bytes are copied before
// returning.
func goString(s C.WasmEdge_String) string {
	if s.Buf == nil || s.Length == 0 {
		return ""
	}
	return C.GoStringN(s.Buf, C.int(s.Length))
}

// wrapBytes gives C a zero-copy view of b. The view is only valid for the
// duration of the C call it is passed to, and every call site MUST call
// runtime.KeepAlive(b) after that C call returns:
//
//	res := C.WasmEdge_LoaderParseFromBytes(cxt, &mod, wrapBytes(b))
//	runtime.KeepAlive(b)
//
// The engine copies the bytes it needs before returning, so the view must
// not be retained by C (this matches the WasmEdge_BytesWrap contract).
func wrapBytes(b []byte) C.WasmEdge_Bytes {
	if len(b) == 0 {
		return C.WasmEdge_Bytes{}
	}
	return C.WasmEdge_BytesWrap((*C.uint8_t)(unsafe.SliceData(b)), C.uint32_t(len(b)))
}

// goBytes copies a C-owned WasmEdge_Bytes into Go memory and frees the C
// buffer. Use for engine-produced buffers the caller owns (for example
// WasmEdge_LoaderSerializeASTModule output).
func goBytesAndFree(b C.WasmEdge_Bytes) []byte {
	if b.Buf == nil || b.Length == 0 {
		return nil
	}
	out := C.GoBytes(unsafe.Pointer(b.Buf), C.int(b.Length))
	C.WasmEdge_BytesDelete(b)
	return out
}

// listStrings drives the C API's two-call list pattern (length query, then
// fill) and copies the resulting borrowed name views into Go strings.
func listStrings(
	length func() C.uint32_t,
	fill func(buf *C.WasmEdge_String, len C.uint32_t) C.uint32_t,
) []string {
	n := length()
	if n == 0 {
		return nil
	}
	buf := make([]C.WasmEdge_String, n)
	got := fill(&buf[0], n)
	if got > n {
		got = n
	}
	out := make([]string, got)
	for i := range out {
		out[i] = goString(buf[i])
	}
	runtime.KeepAlive(buf)
	return out
}

// cStringArray converts a []string into a NULL-free C array of C strings for
// the WASI-style `const char *const *` parameters. The returned release
// function frees every C string; call it via defer.
func cStringArray(ss []string) (**C.char, C.uint32_t, func()) {
	if len(ss) == 0 {
		return nil, 0, func() {}
	}
	arr := make([]*C.char, len(ss))
	for i, s := range ss {
		arr[i] = C.CString(s)
	}
	release := func() {
		for _, p := range arr {
			C.free(unsafe.Pointer(p))
		}
		runtime.KeepAlive(arr)
	}
	return &arr[0], C.uint32_t(len(ss)), release
}

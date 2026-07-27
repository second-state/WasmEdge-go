package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import (
	"fmt"
	"math"
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
//
// Like Go's own allocation primitives, this helper panics when the requested
// string cannot be represented by the native ABI or allocated. Most callers
// are lookup/list APIs whose signatures have no useful recovery path, and
// silently substituting an empty name would address the wrong engine object.
func newWEString(s string) C.WasmEdge_String {
	if len(s) == 0 {
		return C.WasmEdge_String{}
	}
	mustFitWEString(uint64(len(s)))
	out := C.WasmEdge_StringCreateByBuffer(
		(*C.char)(unsafe.Pointer(unsafe.StringData(s))), C.uint32_t(len(s)))
	if out.Buf == nil {
		panic("wasmedge: allocate native string")
	}
	return out
}

func freeWEString(s C.WasmEdge_String) {
	C.WasmEdge_StringDelete(s)
}

func mustFitWEString(length uint64) {
	if length > math.MaxUint32 {
		panic(fmt.Sprintf(
			"wasmedge: string length %d exceeds the native uint32 limit",
			length,
		))
	}
}

// goString copies a WasmEdge_String view into a Go string. Safe on borrowed
// views (names returned by List/Get APIs) because the bytes are copied before
// returning.
func goString(s C.WasmEdge_String) string {
	if s.Buf == nil || s.Length == 0 {
		return ""
	}
	if uint64(s.Length) > uint64(maxInt()) {
		panic("wasmedge: native string is too large for a Go string")
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(s.Buf)), int(s.Length)))
}

// wrapBytes gives C a zero-copy view of b. The view is only valid for the
// duration of the C call it is passed to, and every call site MUST call
// runtime.KeepAlive(b) after that C call returns:
//
//	bytes, err := wrapBytes(b)
//	if err != nil { ... }
//	res := C.WasmEdge_LoaderParseFromBytes(cxt, &mod, bytes)
//	runtime.KeepAlive(b)
//
// The engine copies the bytes it needs before returning, so the view must
// not be retained by C (this matches the WasmEdge_BytesWrap contract).
func wrapBytes(b []byte) (C.WasmEdge_Bytes, error) {
	if uint64(len(b)) > math.MaxUint32 {
		return C.WasmEdge_Bytes{}, fmt.Errorf(
			"WASM binary length %d exceeds uint32: %w", len(b), ErrInvalidArgument)
	}
	if len(b) == 0 {
		return C.WasmEdge_Bytes{}, nil
	}
	return C.WasmEdge_BytesWrap(
		(*C.uint8_t)(unsafe.SliceData(b)), C.uint32_t(len(b)),
	), nil
}

// copyBytes creates C-owned storage for asynchronous native calls that retain
// the byte span after returning to Go.
func copyBytes(b []byte) (C.WasmEdge_Bytes, func(), error) {
	if uint64(len(b)) > math.MaxUint32 {
		return C.WasmEdge_Bytes{}, func() {}, fmt.Errorf(
			"WASM binary length %d exceeds uint32: %w", len(b), ErrInvalidArgument)
	}
	if len(b) == 0 {
		return C.WasmEdge_Bytes{}, func() {}, nil
	}
	copied := C.WasmEdge_BytesCreate(
		(*C.uint8_t)(unsafe.SliceData(b)), C.uint32_t(len(b)))
	runtime.KeepAlive(b)
	if copied.Buf == nil {
		return C.WasmEdge_Bytes{}, func() {}, fmt.Errorf(
			"copy WASM binary: %w", ErrUnavailable)
	}
	return copied, func() { C.WasmEdge_BytesDelete(copied) }, nil
}

// goBytes copies a C-owned WasmEdge_Bytes into Go memory and frees the C
// buffer. Use for engine-produced buffers the caller owns (for example
// WasmEdge_LoaderSerializeASTModule output).
func goBytesAndFree(b C.WasmEdge_Bytes) ([]byte, error) {
	defer C.WasmEdge_BytesDelete(b)
	if b.Buf == nil || b.Length == 0 {
		return nil, nil
	}
	if uint64(b.Length) > uint64(maxInt()) {
		return nil, fmt.Errorf(
			"native byte buffer length %d exceeds Go slice capacity: %w",
			uint64(b.Length), ErrUnavailable,
		)
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(b.Buf)), int(b.Length))
	out := make([]byte, len(src))
	copy(out, src)
	return out, nil
}

// snapshotList drives a two-call list API and retries if the collection grew
// between its length and fill calls. Callers that expose live registries must
// not silently truncate entries added by another goroutine.
func snapshotList[T any](
	length func() uint32,
	fill func([]T) uint32,
) []T {
	n := length()
	for n != 0 {
		if uint64(n) > uint64(maxInt()) {
			panic(fmt.Sprintf(
				"wasmedge: native list length %d exceeds Go slice capacity",
				uint64(n),
			))
		}
		buf := make([]T, int(n))
		got := fill(buf)
		if got <= n {
			return buf[:int(got)]
		}
		n = got
	}
	return nil
}

// snapshotParallelLists is snapshotList for native APIs that fill two
// parallel arrays under one count.
func snapshotParallelLists[A, B any](
	length func() uint32,
	fill func([]A, []B) uint32,
) ([]A, []B) {
	n := length()
	for n != 0 {
		if uint64(n) > uint64(maxInt()) {
			panic(fmt.Sprintf(
				"wasmedge: native list length %d exceeds Go slice capacity",
				uint64(n),
			))
		}
		left := make([]A, int(n))
		right := make([]B, int(n))
		got := fill(left, right)
		if got <= n {
			return left[:int(got)], right[:int(got)]
		}
		n = got
	}
	return nil, nil
}

// listStrings drives the C API's two-call list pattern (length query, then
// fill) and copies the resulting borrowed name views into Go strings.
func listStrings(
	length func() C.uint32_t,
	fill func(buf *C.WasmEdge_String, len C.uint32_t) C.uint32_t,
) []string {
	buf := snapshotList(
		func() uint32 {
			return uint32(length())
		},
		func(buf []C.WasmEdge_String) uint32 {
			return uint32(fill(&buf[0], C.uint32_t(len(buf))))
		},
	)
	out := make([]string, len(buf))
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
	mustFitWEString(uint64(len(ss)))
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

package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"runtime"
	"unsafe"
)

// Memory is a linear memory instance. All offsets and lengths are 64-bit,
// following the 0.17 C API (Memory64 proposal).
type Memory struct {
	ptr  *C.WasmEdge_MemoryInstanceContext
	life lifetime
}

// NewMemory creates a standalone memory instance for exporting from a host
// module. The MemoryType remains caller-owned.
func NewMemory(mt *MemoryType) *Memory {
	if mt == nil {
		return nil
	}
	ptr := C.WasmEdge_MemoryInstanceCreate(mt.ptr)
	runtime.KeepAlive(mt)
	if ptr == nil {
		return nil
	}
	m := &Memory{ptr: ptr}
	arm(m, &m.life, "Memory", func() { C.WasmEdge_MemoryInstanceDelete(ptr) })
	return m
}

func borrowedMemory(ptr *C.WasmEdge_MemoryInstanceContext, owner any) *Memory {
	if ptr == nil {
		return nil
	}
	return &Memory{ptr: ptr, life: borrowed(owner)}
}

// Type returns the memory's type (borrowed).
func (m *Memory) Type() *MemoryType {
	defer runtime.KeepAlive(m)
	return borrowedMemoryType(C.WasmEdge_MemoryInstanceGetMemoryType(m.ptr), m)
}

// Read copies length bytes starting at offset. The engine bounds-checks the
// range and reports an execution error when it is out of bounds.
func (m *Memory) Read(offset, length uint64) ([]byte, error) {
	defer runtime.KeepAlive(m)
	if length == 0 {
		return nil, nil
	}
	buf := make([]byte, length)
	err := newResult(C.WasmEdge_MemoryInstanceGetData(m.ptr,
		(*C.uint8_t)(unsafe.SliceData(buf)), C.uint64_t(offset), C.uint64_t(length)))
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// Write copies data into the memory at offset, bounds-checked by the
// engine.
func (m *Memory) Write(offset uint64, data []byte) error {
	defer runtime.KeepAlive(m)
	if len(data) == 0 {
		return nil
	}
	err := newResult(C.WasmEdge_MemoryInstanceSetData(m.ptr,
		(*C.uint8_t)(unsafe.SliceData(data)), C.uint64_t(offset), C.uint64_t(len(data))))
	runtime.KeepAlive(data)
	return err
}

// UnsafeSlice returns a zero-copy view of the memory range. The view
// aliases engine-owned memory: it is invalidated by memory growth and by
// the instance closing, and writes through it are immediately visible to
// WASM. Prefer Read/Write unless profiling says otherwise.
//
// TODO(intern-hard): A14 — shared memories (Threads proposal) make this
// view concurrently mutable from WASM threads; write the aliasing-rules doc
// block and TestSharedMemory before advertising Threads support (see
// PLAN.md A14 for the acceptance criteria).
func (m *Memory) UnsafeSlice(offset, length uint64) ([]byte, error) {
	defer runtime.KeepAlive(m)
	if length == 0 {
		return nil, nil
	}
	p := C.WasmEdge_MemoryInstanceGetPointer(m.ptr, C.uint64_t(offset), C.uint64_t(length))
	if p == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "memory range out of bounds"}
	}
	return unsafe.Slice((*byte)(p), length), nil
}

// PageCount returns the current size in 64KiB pages.
func (m *Memory) PageCount() uint64 {
	defer runtime.KeepAlive(m)
	return uint64(C.WasmEdge_MemoryInstanceGetPageSize(m.ptr))
}

// GrowPages grows the memory by delta pages.
func (m *Memory) GrowPages(delta uint64) error {
	defer runtime.KeepAlive(m)
	return newResult(C.WasmEdge_MemoryInstanceGrowPage(m.ptr, C.uint64_t(delta)))
}

// Close destroys an owned memory that was not added to a module. No-op for
// borrowed views, after a transfer, and after the first call.
func (m *Memory) Close() error {
	ptr := m.ptr
	return m.life.close(func() { C.WasmEdge_MemoryInstanceDelete(ptr) })
}

package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

// ErrUnsafeSharedMemory reports an attempt to expose shared linear memory as
// a Go slice. Native threads can mutate shared memory outside Go's race
// detector and memory model; use Read and Write instead.
var ErrUnsafeSharedMemory = errors.New("wasmedge: shared memory cannot be exposed as a Go slice")

const wasmPageSize = uint64(64 * 1024)

// Memory is a linear memory instance. All offsets and lengths are 64-bit,
// following the 0.17 C API (Memory64 proposal).
type Memory struct {
	ptr  *C.WasmEdge_MemoryInstanceContext
	life lifetime
}

// NewMemory creates a standalone memory instance for exporting from a host
// module. Invalid descriptors match ErrInvalidArgument. A minimum whose byte
// size cannot be represented on the host, a native allocation failure, or a
// native allocation-size mismatch matches ErrUnavailable.
func NewMemory(mt MemoryType) (*Memory, error) {
	if err := mt.validate(); err != nil {
		return nil, err
	}
	// The largest valid memory64 minimum is 2^48 pages, whose byte size is
	// 2^64 and therefore cannot be represented by the C API's uint64_t byte
	// counts. WasmEdge 0.17.1 otherwise creates a zero-page instance.
	if mt.Limits.Min > ^uint64(0)/wasmPageSize {
		return nil, fmt.Errorf(
			"memory minimum %d pages has an unrepresentable byte size: %w",
			mt.Limits.Min, ErrUnavailable,
		)
	}
	cmt, freeType, err := mt.build()
	if err != nil {
		return nil, err
	}
	defer freeType()
	ptr := C.WasmEdge_MemoryInstanceCreate(cmt)
	if ptr == nil {
		return nil, fmt.Errorf("create memory: %w", ErrUnavailable)
	}
	pages := uint64(C.WasmEdge_MemoryInstanceGetPageSize(ptr))
	if err := validateMemoryPageCount(mt.Limits.Min, pages); err != nil {
		C.WasmEdge_MemoryInstanceDelete(ptr)
		return nil, err
	}
	m := &Memory{ptr: ptr}
	arm(m, &m.life, "Memory", func() { C.WasmEdge_MemoryInstanceDelete(ptr) })
	return m, nil
}

func validateMemoryPageCount(requested, allocated uint64) error {
	if allocated != requested {
		return fmt.Errorf(
			"create memory allocated %d pages, want %d: %w",
			allocated, requested, ErrUnavailable,
		)
	}
	return nil
}

func borrowedMemory(ptr *C.WasmEdge_MemoryInstanceContext, owner any) *Memory {
	if ptr == nil {
		return nil
	}
	return &Memory{ptr: ptr, life: borrowed(owner)}
}

func (m *Memory) assertAlive() { m.life.assertAlive("Memory") }

func (m *Memory) acquireLease() (func(), error) {
	return m.life.acquire("Memory")
}

// Type returns a copied descriptor of the memory's type.
func (m *Memory) Type() MemoryType {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	mt, _ := memoryTypeFromC(C.WasmEdge_MemoryInstanceGetMemoryType(m.ptr))
	return mt
}

// Read copies length bytes starting at offset. It rejects ranges outside the
// current memory before allocating and reports lengths that cannot fit in a
// Go slice as ErrInvalidArgument.
func (m *Memory) Read(offset, length uint64) ([]byte, error) {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	if length > uint64(maxInt()) {
		return nil, fmt.Errorf(
			"memory read length %d exceeds Go slice capacity: %w",
			length, ErrInvalidArgument,
		)
	}
	if err := m.checkRange(offset, length); err != nil {
		return nil, err
	}
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

// Write copies data into the memory at offset. The binding checks the complete
// range before entering the engine.
func (m *Memory) Write(offset uint64, data []byte) error {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	if err := m.checkRange(offset, uint64(len(data))); err != nil {
		return err
	}
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
// WASM. Prefer Read/Write unless profiling says otherwise. Shared memories
// are rejected because native threads can access them outside Go's memory
// model and race detector.
func (m *Memory) UnsafeSlice(offset, length uint64) ([]byte, error) {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	if m.Type().Limits.Shared {
		return nil, ErrUnsafeSharedMemory
	}
	if length > uint64(maxInt()) {
		return nil, fmt.Errorf(
			"memory view length %d exceeds Go slice capacity: %w",
			length, ErrInvalidArgument,
		)
	}
	if err := m.checkRange(offset, length); err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}
	p := C.WasmEdge_MemoryInstanceGetPointer(m.ptr, C.uint64_t(offset), C.uint64_t(length))
	if p == nil {
		return nil, memoryOutOfBoundsError()
	}
	return unsafe.Slice((*byte)(p), length), nil
}

func (m *Memory) checkRange(offset, length uint64) error {
	pages := uint64(C.WasmEdge_MemoryInstanceGetPageSize(m.ptr))
	if pages > ^uint64(0)/wasmPageSize {
		// The represented memory is at least the full uint64 address space;
		// accept a range whose exclusive end is exactly 2^64, but reject one
		// that would extend beyond it.
		if length != 0 && length-1 > ^uint64(0)-offset {
			return memoryOutOfBoundsError()
		}
		return nil
	}
	size := pages * wasmPageSize
	if offset > size || length > size-offset {
		return memoryOutOfBoundsError()
	}
	return nil
}

func memoryOutOfBoundsError() error {
	return &Error{
		Category: ErrCategoryWASM,
		Code:     ErrCodeMemoryOutOfBounds,
		Message:  "memory range out of bounds",
	}
}

// PageCount returns the current size in 64KiB pages.
func (m *Memory) PageCount() uint64 {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	return uint64(C.WasmEdge_MemoryInstanceGetPageSize(m.ptr))
}

// GrowPages grows the memory by delta pages.
func (m *Memory) GrowPages(delta uint64) error {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	return newResult(C.WasmEdge_MemoryInstanceGrowPage(m.ptr, C.uint64_t(delta)))
}

// Close destroys an owned memory that was not added to a module. No-op for
// borrowed views, after a transfer, and after the first call.
func (m *Memory) Close() error {
	ptr := m.ptr
	return m.life.close(func() { C.WasmEdge_MemoryInstanceDelete(ptr) })
}

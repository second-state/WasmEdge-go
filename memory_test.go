package wasmedge

import (
	"bytes"
	"errors"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

// instantiateMemoryModule returns the instantiated Memory fixture and its
// exported memory.
func instantiateMemoryModule(t *testing.T) (*Executor, *Module, *Memory) {
	t.Helper()
	loader, _ := NewLoader(nil)
	t.Cleanup(func() { loader.Close() })
	ast, err := loader.LoadBytes(testwasm.MemoryModule())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ast.Close() })
	validator, _ := NewValidator(nil)
	t.Cleanup(func() { validator.Close() })
	if err := validator.Validate(ast); err != nil {
		t.Fatal(err)
	}
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	inst, err := exec.Instantiate(store, ast)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		inst.Close()
		store.Close()
		exec.Close()
	})
	mem, ok := inst.Memory("mem")
	if !ok {
		t.Fatal("mem export missing")
	}
	return exec, inst, mem
}

func TestMemoryReadWrite(t *testing.T) {
	exec, inst, mem := instantiateMemoryModule(t)

	got, err := mem.Read(0, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte("hello")) {
		t.Fatalf("data segment: %q", got)
	}

	// The exported load8 must see the same bytes the Go side sees.
	fn, _ := inst.Function("load8")
	out, err := exec.Invoke(fn, I32(1))
	if err != nil {
		t.Fatal(err)
	}
	if out[0].I32() != int32('e') {
		t.Fatalf("load8(1) = %d", out[0].I32())
	}

	if err := mem.Write(10, []byte{0xAB}); err != nil {
		t.Fatal(err)
	}
	out, err = exec.Invoke(fn, I32(10))
	if err != nil {
		t.Fatal(err)
	}
	if out[0].I32() != 0xAB {
		t.Fatalf("write not visible to wasm: %d", out[0].I32())
	}
}

func TestMemoryBounds(t *testing.T) {
	_, _, mem := instantiateMemoryModule(t)

	var we *Error
	if _, err := mem.Read(65536-2, 4); !errors.As(err, &we) {
		t.Fatalf("out-of-bounds read must fail, got %v", err)
	}
	if err := mem.Write(65536, []byte{1}); !errors.As(err, &we) {
		t.Fatalf("out-of-bounds write must fail, got %v", err)
	}
	if _, err := mem.UnsafeSlice(65530, 100); !errors.As(err, &we) {
		t.Fatalf("out-of-bounds view must fail, got %v", err)
	}
	if _, err := mem.Read(0, ^uint64(0)); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("unrepresentable read length: got %v, want ErrInvalidArgument", err)
	}
	if _, err := mem.UnsafeSlice(0, ^uint64(0)); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("unrepresentable view length: got %v, want ErrInvalidArgument", err)
	}

	for name, call := range map[string]func(uint64) error{
		"read": func(offset uint64) error {
			_, err := mem.Read(offset, 0)
			return err
		},
		"write": func(offset uint64) error {
			return mem.Write(offset, nil)
		},
		"unsafe slice": func(offset uint64) error {
			_, err := mem.UnsafeSlice(offset, 0)
			return err
		},
	} {
		t.Run("zero length "+name, func(t *testing.T) {
			if err := call(65536); err != nil {
				t.Fatalf("offset at end: %v", err)
			}
			if err := call(65537); !errors.As(err, &we) ||
				we.Code != ErrCodeMemoryOutOfBounds {
				t.Fatalf("offset past end: got %v, want ErrCodeMemoryOutOfBounds", err)
			}
		})
	}
}

func TestMemoryGrowAndUnsafeSlice(t *testing.T) {
	_, _, mem := instantiateMemoryModule(t)

	if got := mem.PageCount(); got != 1 {
		t.Fatalf("pages: %d", got)
	}
	if lim := mem.Type().Limits; lim.Min != 1 {
		t.Fatalf("type limits before grow: %+v", lim)
	}
	view, err := mem.UnsafeSlice(0, 5)
	if err != nil {
		t.Fatal(err)
	}
	if string(view) != "hello" {
		t.Fatalf("view: %q", view)
	}
	view[0] = 'H' // writes through the view are visible to the engine
	got, _ := mem.Read(0, 5)
	if string(got) != "Hello" {
		t.Fatalf("write-through failed: %q", got)
	}

	if err := mem.GrowPages(1); err != nil {
		t.Fatal(err)
	}
	if got := mem.PageCount(); got != 2 {
		t.Fatalf("pages after grow: %d", got)
	}
	// NOTE: `view` is invalid from here on (growth may move the buffer) —
	// that is exactly the lifetime rule UnsafeSlice documents.

	// The engine folds the current size back into the type's minimum.
	if lim := mem.Type().Limits; lim.Min != 2 {
		t.Fatalf("type limits after grow: %+v", lim)
	}
}

func TestUnsafeSliceRejectsSharedMemory(t *testing.T) {
	mem, err := NewMemory(MemoryType{
		Limits: Limits{Min: 1, Max: 1, HasMax: true, Shared: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer mem.Close()

	if _, err = mem.UnsafeSlice(0, 1); !errors.Is(err, ErrUnsafeSharedMemory) {
		t.Fatalf("UnsafeSlice on shared memory: %v", err)
	}
	if err := mem.Write(0, []byte{42}); err != nil {
		t.Fatal(err)
	}
	got, err := mem.Read(0, 1)
	if err != nil || len(got) != 1 || got[0] != 42 {
		t.Fatalf("safe shared-memory access: got=%v err=%v", got, err)
	}
}

func TestNewMemoryRejectsUnrepresentableAndIncompleteAllocations(t *testing.T) {
	if _, err := NewMemory(MemoryType{Limits: Limits{
		Min:  maxMemory64Pages,
		Is64: true,
	}}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("2^48-page memory64: got %v, want ErrUnavailable", err)
	}

	if err := validateMemoryPageCount(1, 0); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("incomplete native allocation: got %v, want ErrUnavailable", err)
	}
	if err := validateMemoryPageCount(1, 1); err != nil {
		t.Fatalf("complete native allocation: %v", err)
	}
}

package wasmedge

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestCompilerAOT(t *testing.T) {
	comp, err := NewCompiler(&Config{
		Compiler: CompilerConfig{OutputFormat: OutputFormatWasm},
	})
	if err != nil {
		var we *Error
		if errors.As(err, &we) && we.Code == ErrCodeAOTDisabled {
			t.Skip("library built without the AOT backend")
		}
		t.Fatal(err)
	}
	defer comp.Close()

	out := filepath.Join(t.TempDir(), "fib_aot.wasm")
	if err := comp.CompileBytes(testwasm.FibModule(), out); err != nil {
		t.Fatal(err)
	}

	// The universal-WASM artifact must run and produce identical results.
	// (Staged path; the RunFile convenience is intern task A4.)
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	if err := vm.LoadFile(out); err != nil {
		t.Fatal(err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatal(err)
	}
	res, err := vm.Execute("fib", I32(10))
	if err != nil {
		t.Fatal(err)
	}
	if res[0].I32() != 89 {
		t.Fatalf("AOT fib(10) = %v", res)
	}
}

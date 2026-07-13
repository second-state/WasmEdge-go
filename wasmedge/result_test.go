package wasmedge

import "testing"

func TestResult(t *testing.T) {
	if Result_Success.Error() != "success" {
		t.Errorf("success message = %q", Result_Success.Error())
	}
	if Result_Terminate.Error() != "terminated" {
		t.Errorf("terminated message = %q", Result_Terminate.Error())
	}

	res := NewResult(ErrCategory_UserLevel, 5958)
	if res.GetErrorCategory() != ErrCategory_UserLevel {
		t.Errorf("category = %d, want user-level", res.GetErrorCategory())
	}
	if res.GetCode() != 5958 {
		t.Errorf("code = %d, want 5958", res.GetCode())
	}

	wasmres := NewResult(ErrCategory_WASM, 1)
	if wasmres.GetErrorCategory() != ErrCategory_WASM {
		t.Errorf("category = %d, want WASM", wasmres.GetErrorCategory())
	}
}

func TestResultFromExecution(t *testing.T) {
	vm := NewVM()
	defer vm.Release()

	_, err := vm.RunWasmFile(testWasmPath("trap.wasm"), "trap")
	if err == nil {
		t.Fatal("trap should fail")
	}
	res := err.(*Result)
	if res.GetErrorCategory() != ErrCategory_WASM {
		t.Error("trap should be a WASM-category error")
	}
	if res.Error() != "unreachable" {
		t.Errorf("trap message = %q, want \"unreachable\"", res.Error())
	}
}

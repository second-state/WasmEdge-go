package wasmedge

import "testing"

func TestValTypes(t *testing.T) {
	cases := []struct {
		vt   *ValType
		str  string
		want func(*ValType) bool
	}{
		{NewValTypeI32(), "i32", (*ValType).IsI32},
		{NewValTypeI64(), "i64", (*ValType).IsI64},
		{NewValTypeF32(), "f32", (*ValType).IsF32},
		{NewValTypeF64(), "f64", (*ValType).IsF64},
		{NewValTypeV128(), "v128", (*ValType).IsV128},
		{NewValTypeFuncRef(), "funcref", (*ValType).IsFuncRef},
		{NewValTypeExternRef(), "externref", (*ValType).IsExternRef},
	}
	for _, c := range cases {
		if !c.want(c.vt) {
			t.Errorf("predicate for %q returned false", c.str)
		}
		if c.vt.String() != c.str {
			t.Errorf("String() = %q, want %q", c.vt.String(), c.str)
		}
		if !c.vt.IsEqual(c.vt) {
			t.Errorf("%q should be equal to itself", c.str)
		}
	}
	if NewValTypeI32().IsEqual(NewValTypeI64()) {
		t.Error("i32 should not equal i64")
	}
	if !NewValTypeFuncRef().IsRef() || !NewValTypeExternRef().IsRef() {
		t.Error("funcref/externref should be reference types")
	}
	if NewValTypeI32().IsRef() {
		t.Error("i32 should not be a reference type")
	}
	if !NewValTypeFuncRef().IsRefNull() {
		t.Error("plain funcref should be nullable")
	}
}

func TestValMut(t *testing.T) {
	if ValMut_Const.String() != "const" || ValMut_Var.String() != "var" {
		t.Error("wrong ValMut strings")
	}
}

func TestV128(t *testing.T) {
	const high, low = uint64(0x0123456789abcdef), uint64(0xfedcba9876543210)
	val := NewV128(high, low)
	gotHigh, gotLow := val.GetVal()
	if gotHigh != high || gotLow != low {
		t.Errorf("V128 round-trip mismatch: got (%x, %x), want (%x, %x)",
			gotHigh, gotLow, high, low)
	}
}

func TestV128ThroughWasm(t *testing.T) {
	vm := NewVM()
	defer vm.Release()

	const high, low = uint64(0xdeadbeefcafebabe), uint64(0x0102030405060708)
	rets, err := vm.RunWasmFile(testWasmPath("refs.wasm"), "v128_id", NewV128(high, low))
	if err != nil {
		t.Fatalf("v128_id failed: %v", err)
	}
	got, ok := rets[0].(V128)
	if !ok {
		t.Fatalf("v128_id returned %T, want V128", rets[0])
	}
	gotHigh, gotLow := got.GetVal()
	if gotHigh != high || gotLow != low {
		t.Errorf("v128 through wasm mismatch: got (%x, %x)", gotHigh, gotLow)
	}

	rets, err = vm.Execute("splat_add", int32(30), int32(12))
	if err != nil {
		t.Fatalf("splat_add failed: %v", err)
	}
	if rets[0].(int32) != 42 {
		t.Errorf("splat_add = %v, want 42", rets[0])
	}
}

func TestExternRef(t *testing.T) {
	vm := NewVM()
	defer vm.Release()

	type payload struct{ answer int }
	data := &payload{answer: 42}
	ref := NewExternRef(data)

	rets, err := vm.RunWasmFile(testWasmPath("refs.wasm"), "extern_id", ref)
	if err != nil {
		t.Fatalf("extern_id failed: %v", err)
	}
	got, ok := rets[0].(ExternRef)
	if !ok {
		t.Fatalf("extern_id returned %T, want ExternRef", rets[0])
	}
	if got.IsNull() {
		t.Error("returned externref should not be null")
	}
	if got.GetRef().(*payload).answer != 42 {
		t.Error("externref payload mismatch")
	}

	rets, err = vm.Execute("is_null_extern", ref)
	if err != nil {
		t.Fatalf("is_null_extern failed: %v", err)
	}
	if rets[0].(int32) != 0 {
		t.Error("live externref should not be ref.null")
	}

	// After releasing, the reference cannot be used as an argument anymore.
	ref.Release()
	if ref.GetRef() != nil {
		t.Error("released externref should return nil")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("using a released externref should panic")
			}
		}()
		_, _ = vm.Execute("extern_id", ref)
	}()
}

func TestFuncRef(t *testing.T) {
	// Instantiate types.wasm which exports a funcref table with slot 0
	// initialized and the other slots null.
	conf := NewConfigure()
	defer conf.Release()
	store := NewStore()
	defer store.Release()
	executor := NewExecutorWithConfig(conf)
	defer executor.Release()

	env := buildEnvModule(t, 666)
	defer env.Release()
	if err := executor.RegisterImport(store, env); err != nil {
		t.Fatalf("RegisterImport failed: %v", err)
	}
	mod := instantiateFile(t, executor, store, testWasmPath("types.wasm"))
	defer mod.Release()

	tab := mod.FindTable("tab")
	if tab == nil {
		t.Fatal("table export not found")
	}

	data, err := tab.GetData(0)
	if err != nil {
		t.Fatalf("Table.GetData(0) failed: %v", err)
	}
	fref, ok := data.(FuncRef)
	if !ok {
		t.Fatalf("table slot 0 is %T, want FuncRef", data)
	}
	if fref.IsNull() {
		t.Error("table slot 0 should hold a non-null funcref")
	}
	fn := fref.GetRef()
	if fn == nil {
		t.Fatal("funcref should reference a function instance")
	}
	rets, err := executor.Invoke(fn, int32(20), int32(22))
	if err != nil {
		t.Fatalf("invoking table function failed: %v", err)
	}
	if rets[0].(int32) != 42 {
		t.Errorf("table function = %v, want 42", rets[0])
	}

	data, err = tab.GetData(1)
	if err != nil {
		t.Fatalf("Table.GetData(1) failed: %v", err)
	}
	fref, ok = data.(FuncRef)
	if !ok {
		t.Fatalf("table slot 1 is %T, want FuncRef", data)
	}
	if !fref.IsNull() {
		t.Error("table slot 1 should be a null funcref")
	}
	if fref.GetRef() != nil {
		t.Error("null funcref should have no function instance")
	}
}

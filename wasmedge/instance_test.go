package wasmedge

import (
	"bytes"
	"testing"
)

func TestModuleCreation(t *testing.T) {
	mod := NewModule("host")
	if mod == nil {
		t.Fatal("NewModule() returned nil")
	}
	defer mod.Release()
	if mod.GetName() != "host" {
		t.Errorf("module name = %q, want \"host\"", mod.GetName())
	}

	ftype := NewFunctionType(nil, nil)
	fn := NewFunction(ftype, func(interface{}, *CallingFrame, []interface{}) ([]interface{}, Result) {
		return nil, Result_Success
	}, nil, 0)
	ftype.Release()
	mod.AddFunction("f", fn)

	tlim := NewLimitWithMax(2, 4)
	ttype := NewTableType(NewValTypeFuncRef(), tlim)
	tlim.Release()
	tab := NewTable(ttype)
	ttype.Release()
	if tab == nil {
		t.Fatal("NewTable() returned nil")
	}
	mod.AddTable("t", tab)

	mlim := NewLimitWithMax(1, 2)
	mtype := NewMemoryType(mlim)
	mlim.Release()
	mem := NewMemory(mtype)
	mtype.Release()
	if mem == nil {
		t.Fatal("NewMemory() returned nil")
	}
	mod.AddMemory("m", mem)

	gtype := NewGlobalType(NewValTypeI64(), ValMut_Var)
	glob := NewGlobal(gtype, int64(-99))
	gtype.Release()
	if glob == nil {
		t.Fatal("NewGlobal() returned nil")
	}
	mod.AddGlobal("g", glob)

	if got := mod.ListFunction(); len(got) != 1 || got[0] != "f" {
		t.Errorf("ListFunction() = %v, want [f]", got)
	}
	if got := mod.ListTable(); len(got) != 1 || got[0] != "t" {
		t.Errorf("ListTable() = %v, want [t]", got)
	}
	if got := mod.ListMemory(); len(got) != 1 || got[0] != "m" {
		t.Errorf("ListMemory() = %v, want [m]", got)
	}
	if got := mod.ListGlobal(); len(got) != 1 || got[0] != "g" {
		t.Errorf("ListGlobal() = %v, want [g]", got)
	}
	if got := mod.ListTag(); len(got) != 0 {
		t.Errorf("ListTag() = %v, want empty", got)
	}

	if mod.FindFunction("f") == nil || mod.FindTable("t") == nil ||
		mod.FindMemory("m") == nil || mod.FindGlobal("g") == nil {
		t.Error("Find* should locate the added instances")
	}
	if mod.FindFunction("nope") != nil || mod.FindTable("nope") != nil ||
		mod.FindMemory("nope") != nil || mod.FindGlobal("nope") != nil ||
		mod.FindTag("nope") != nil {
		t.Error("Find* should return nil for unknown names")
	}
}

func TestFunctionInstance(t *testing.T) {
	ftype := NewFunctionType(
		[]*ValType{NewValTypeI32()},
		[]*ValType{NewValTypeI32(), NewValTypeI32()})
	fn := NewFunction(ftype, func(data interface{}, frame *CallingFrame, params []interface{}) ([]interface{}, Result) {
		v := params[0].(int32)
		return []interface{}{v, v + 1}, Result_Success
	}, nil, 0)
	ftype.Release()
	if fn == nil {
		t.Fatal("NewFunction() returned nil")
	}
	defer fn.Release()

	got := fn.GetFunctionType()
	if got.GetParametersLength() != 1 || got.GetReturnsLength() != 2 {
		t.Error("function type mismatch")
	}

	if NewFunction(nil, nil, nil, 0) != nil {
		t.Error("NewFunction(nil type) should return nil")
	}
}

func TestTableInstance(t *testing.T) {
	tlim := NewLimitWithMax(4, 8)
	ttype := NewTableType(NewValTypeFuncRef(), tlim)
	tlim.Release()
	tab := NewTable(ttype)
	ttype.Release()
	if tab == nil {
		t.Fatal("NewTable() returned nil")
	}
	defer tab.Release()

	if tab.GetSize() != 4 {
		t.Errorf("table size = %d, want 4", tab.GetSize())
	}
	lim := tab.GetTableType().GetLimit()
	if lim.GetMin() != 4 || lim.GetMax() != 8 {
		t.Errorf("table type limit mismatch: %+v", lim)
	}

	if err := tab.Grow(3); err != nil {
		t.Errorf("Grow(3) failed: %v", err)
	}
	if tab.GetSize() != 7 {
		t.Errorf("table size after grow = %d, want 7", tab.GetSize())
	}
	if err := tab.Grow(10); err == nil {
		t.Error("growing beyond the max should fail")
	}

	// Uninitialized slots hold null funcref values.
	data, err := tab.GetData(0)
	if err != nil {
		t.Fatalf("GetData(0) failed: %v", err)
	}
	if fref, ok := data.(FuncRef); !ok || !fref.IsNull() {
		t.Errorf("uninitialized slot should be a null funcref, got %T", data)
	}
	if _, err := tab.GetData(100); err == nil {
		t.Error("out-of-bounds access should fail")
	}
	if err := tab.SetData(int32(1), 0); err == nil {
		t.Error("setting a non-reference value should fail")
	}
}

func TestMemoryInstance(t *testing.T) {
	mlim := NewLimitWithMax(1, 3)
	mtype := NewMemoryType(mlim)
	mlim.Release()
	mem := NewMemory(mtype)
	mtype.Release()
	if mem == nil {
		t.Fatal("NewMemory() returned nil")
	}
	defer mem.Release()

	if mem.GetPageSize() != 1 {
		t.Errorf("page size = %d, want 1", mem.GetPageSize())
	}

	payload := []byte("wasmedge-go")
	if err := mem.SetData(payload, 100, uint(len(payload))); err != nil {
		t.Fatalf("SetData failed: %v", err)
	}
	got, err := mem.GetData(100, uint(len(payload)))
	if err != nil {
		t.Fatalf("GetData failed: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("GetData = %q, want %q", got, payload)
	}

	// GetData returns a view into the WASM memory: writes through it are
	// visible on subsequent reads.
	got[0] = 'W'
	again, err := mem.GetData(100, 1)
	if err != nil {
		t.Fatal(err)
	}
	if again[0] != 'W' {
		t.Error("GetData should be a view into the linear memory")
	}

	if err := mem.GrowPage(1); err != nil {
		t.Errorf("GrowPage(1) failed: %v", err)
	}
	if mem.GetPageSize() != 2 {
		t.Errorf("page size after grow = %d, want 2", mem.GetPageSize())
	}
	if err := mem.GrowPage(5); err == nil {
		t.Error("growing beyond the max should fail")
	}
	if _, err := mem.GetData(65536*2-1, 2); err == nil {
		t.Error("out-of-bounds GetData should fail")
	}
	if err := mem.SetData(payload, 65536*2-1, uint(len(payload))); err == nil {
		t.Error("out-of-bounds SetData should fail")
	}
}

func TestGlobalInstance(t *testing.T) {
	gtype := NewGlobalType(NewValTypeI64(), ValMut_Var)
	glob := NewGlobal(gtype, int64(1000))
	gtype.Release()
	if glob == nil {
		t.Fatal("NewGlobal() returned nil")
	}
	defer glob.Release()

	if glob.GetValue().(int64) != 1000 {
		t.Errorf("global value = %v, want 1000", glob.GetValue())
	}
	if err := glob.SetValue(int64(-42)); err != nil {
		t.Errorf("SetValue failed: %v", err)
	}
	if glob.GetValue().(int64) != -42 {
		t.Errorf("global value = %v, want -42", glob.GetValue())
	}
	if err := glob.SetValue(int32(1)); err == nil {
		t.Error("setting a value of the wrong type should fail")
	}

	ctype := NewGlobalType(NewValTypeI32(), ValMut_Const)
	cglob := NewGlobal(ctype, int32(7))
	ctype.Release()
	defer cglob.Release()
	if err := cglob.SetValue(int32(8)); err == nil {
		t.Error("setting a const global should fail")
	}
}

func TestWasiModule(t *testing.T) {
	wasi := NewWasiModule(
		[]string{"prog", "arg1"},
		[]string{"KEY=VALUE"},
		[]string{".:."})
	if wasi == nil {
		t.Fatal("NewWasiModule() returned nil")
	}
	defer wasi.Release()

	if wasi.GetName() != "wasi_snapshot_preview1" {
		t.Errorf("WASI module name = %q", wasi.GetName())
	}
	if len(wasi.ListFunction()) == 0 {
		t.Error("WASI module should export functions")
	}
	wasi.InitWasi([]string{"prog"}, []string{}, []string{})

	wasiFds := NewWasiModuleWithFds([]string{"prog"}, nil, nil, 0, 1, 2)
	if wasiFds == nil {
		t.Fatal("NewWasiModuleWithFds() returned nil")
	}
	defer wasiFds.Release()
	wasiFds.InitWasiWithFds([]string{"prog"}, nil, nil, 0, 1, 2)
}

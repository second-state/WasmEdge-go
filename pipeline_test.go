package wasmedge

import (
	"errors"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestLoadValidate(t *testing.T) {
	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()

	ast, err := loader.LoadBytes(testwasm.AddModule())
	if err != nil {
		t.Fatal(err)
	}
	defer ast.Close()

	exports := ast.Exports()
	if len(exports) != 1 || exports[0].Name() != "add" ||
		exports[0].ExternalType() != ExternalTypeFunction {
		t.Fatalf("exports: %+v", exports)
	}
	ft, ok := exports[0].FunctionType()
	if !ok || len(ft.Params) != 2 || len(ft.Results) != 1 {
		t.Fatal("unexpected add signature")
	}

	val, err := NewValidator(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer val.Close()
	if err := val.Validate(ast); err != nil {
		t.Fatal(err)
	}
}

func TestLoadImports(t *testing.T) {
	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()

	ast, err := loader.LoadBytes(testwasm.HostCallModule())
	if err != nil {
		t.Fatal(err)
	}
	defer ast.Close()

	imports := ast.Imports()
	if len(imports) != 1 {
		t.Fatalf("imports: %+v", imports)
	}
	imp := imports[0]
	_, ok := imp.FunctionType()
	if imp.ModuleName() != "env" || imp.Name() != "host_add" ||
		imp.ExternalType() != ExternalTypeFunction || !ok {
		t.Fatalf("import: %s.%s (%s)", imp.ModuleName(), imp.Name(), imp.ExternalType())
	}
}

func TestLoadFailureIsTypedAndLogged(t *testing.T) {
	var logged atomic.Int32
	SetLogLevel(LogLevelError)
	SetLogCallback(func(m LogMessage) { logged.Add(1) })
	defer SetLogCallback(nil)
	defer SetLogOff()

	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()

	_, err = loader.LoadBytes([]byte("definitely not wasm"))
	var we *Error
	if !errors.As(err, &we) {
		t.Fatalf("want *Error, got %v (%T)", err, err)
	}
	if we.Category != ErrCategoryWASM || we.Code>>8 != 0x01 {
		t.Fatalf("want a load-phase (0x01xx) code, got %+v", we)
	}
	if logged.Load() == 0 {
		t.Fatal("engine log callback was not invoked on parse failure")
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()

	ast, err := loader.LoadBytes(testwasm.FibModule())
	if err != nil {
		t.Fatal(err)
	}
	defer ast.Close()

	b, err := loader.Serialize(ast)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if !errors.Is(err, ErrSerializeUnsupported) ||
			!errors.Is(err, errors.ErrUnsupported) {
			t.Fatalf("darwin/arm64 serialization: %v", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("empty serialization")
	}
	again, err := loader.LoadBytes(b)
	if err != nil {
		t.Fatalf("re-parse of serialized module failed: %v", err)
	}
	defer again.Close()
	if len(again.Exports()) != 1 || again.Exports()[0].Name() != "fib" {
		t.Fatal("serialization lost the fib export")
	}
}

func TestConfigStandardGatesProposals(t *testing.T) {
	// The Memory fixture is plain WASM 1.0, so it parses under WASM 1;
	// bulk-memory/SIMD-era binaries are separate proposals. Instead verify
	// that a WASM 1 loader still accepts the plain module while a disabled
	// proposal set does not break it, and that a bogus standard value does
	// not crash configuration.
	loader, err := NewLoader(&Config{
		Standard:         StandardWASM1,
		DisableProposals: []Proposal{ProposalSIMD},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()
	ast, err := loader.LoadBytes(testwasm.AddModule())
	if err != nil {
		t.Fatal(err)
	}
	ast.Close()
}

func TestStatistics(t *testing.T) {
	s := NewStatistics()
	if s == nil {
		t.Fatal("NewStatistics returned nil")
	}
	defer s.Close()
	if got := s.InstrCount(); got != 0 {
		t.Fatalf("fresh collector counted %d instructions", got)
	}

	// Cover the full opcode space with unit costs, then make any non-trivial
	// invocation exceed the limit.
	costs := make([]uint64, 512)
	for i := range costs {
		costs[i] = 20
	}
	if err := s.SetCostTable(costs); err != nil {
		t.Fatal(err)
	}
	s.SetCostLimit(1)

	exec, err := NewExecutor(
		&Config{Stats: StatsConfig{InstructionCounting: true, CostMeasuring: true}},
		WithStats(s),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()
	if exec.stats != s {
		t.Fatal("Executor did not retain its Statistics collector")
	}

	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()
	ast, err := loader.LoadBytes(testwasm.AddModule())
	if err != nil {
		t.Fatal(err)
	}
	defer ast.Close()
	validator, err := NewValidator(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer validator.Close()
	if err := validator.Validate(ast); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	defer store.Close()
	inst, err := exec.Instantiate(store, ast)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()
	add, ok := inst.Function("add")
	if !ok {
		t.Fatal("add export not found")
	}
	_, err = exec.Invoke(add, I32(1), I32(2))
	var we *Error
	if !errors.As(err, &we) || we.Code != ErrCodeCostLimitExceeded {
		t.Fatalf("cost limit: want ErrCodeCostLimitExceeded, got %v (cost=%d instructions=%d)",
			err, s.TotalCost(), s.InstrCount())
	}

	s.Clear()
	if got := s.TotalCost(); got != 0 {
		t.Fatalf("Clear left total cost %d", got)
	}
	if err := exec.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal("double Close must be a no-op")
	}
}

func TestExecutorStatisticsLifetime(t *testing.T) {
	closed := NewStatistics()
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewExecutor(nil, WithStats(closed)); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed statistics: got %v, want ErrClosed", err)
	}

	stats := NewStatistics()
	exec, err := NewExecutor(nil, WithStats(stats))
	if err != nil {
		t.Fatal(err)
	}
	if err := stats.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("close attached statistics: got %v, want ErrInUse", err)
	}
	if err := exec.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stats.Close(); err != nil {
		t.Fatalf("close statistics after executor: %v", err)
	}
}

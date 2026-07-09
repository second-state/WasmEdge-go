package wasmedge

import "testing"

func TestStatistics(t *testing.T) {
	conf := NewConfigure()
	defer conf.Release()
	conf.SetStatisticsInstructionCounting(true)
	conf.SetStatisticsTimeMeasuring(true)
	conf.SetStatisticsCostMeasuring(true)

	stat := NewStatistics()
	if stat == nil {
		t.Fatal("NewStatistics() returned nil")
	}
	defer stat.Release()

	costtable := make([]uint64, 512)
	for i := range costtable {
		costtable[i] = 1
	}
	stat.SetCostTable(costtable)
	stat.SetCostLimit(1000000)

	store := NewStore()
	defer store.Release()
	executor := NewExecutorWithConfigAndStatistics(conf, stat)
	if executor == nil {
		t.Fatal("NewExecutorWithConfigAndStatistics() returned nil")
	}
	defer executor.Release()

	mod := instantiateFile(t, executor, store, testWasmPath("fib.wasm"))
	defer mod.Release()

	if _, err := executor.Invoke(mod.FindFunction("fib"), int32(10)); err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if stat.GetInstrCount() == 0 {
		t.Error("instruction count should be positive")
	}
	if stat.GetInstrPerSecond() <= 0 {
		t.Error("instructions per second should be positive")
	}
	if stat.GetTotalCost() == 0 {
		t.Error("total cost should be positive")
	}

	// A tiny cost limit interrupts the execution.
	stat.SetCostLimit(1)
	if _, err := executor.Invoke(mod.FindFunction("fib"), int32(20)); err == nil {
		t.Error("execution should fail when exceeding the cost limit")
	}
}

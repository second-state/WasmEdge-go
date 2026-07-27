package wasmedge

import (
	"errors"
	"testing"
	"time"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestExecutorInvokeAsync(t *testing.T) {
	background := make(chan bool, 1)
	host := MustWrapFunc(func(call *CallContext, a, b int32) int32 {
		background <- call.Context().Done() == nil
		return a + b
	})
	executor, instance := instantiateWithHost(t, host)
	fn, ok := instance.Function("call_host")
	if !ok {
		t.Fatal("call_host export missing")
	}

	execution, err := executor.InvokeAsync(fn, I32(20), I32(22))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := executor.InvokeAsync(fn, I32(1), I32(2)); !errors.Is(err, ErrInUse) {
		t.Fatalf("overlapping invocation: got %v, want ErrInUse", err)
	}
	results, err := execution.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].I32() != 42 {
		t.Fatalf("results: %v", results)
	}
	if !<-background {
		t.Fatal("manual async callback did not receive context.Background")
	}

	// The terminal Wait releases the Executor gate.
	next, err := executor.InvokeAsync(fn, I32(2), I32(3))
	if err != nil {
		t.Fatalf("invoke after Wait: %v", err)
	}
	results, err = next.Wait()
	if err != nil || len(results) != 1 || results[0].I32() != 5 {
		t.Fatalf("second results: %v, err=%v", results, err)
	}
	<-background

	if _, err := executor.InvokeAsync(nil); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("nil function: got %v, want ErrInvalidArgument", err)
	}
}

func TestExecutionCancelMayRaceWait(t *testing.T) {
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	instantiateVMFixture(t, vm, testwasm.LoopModule())

	for i := 0; i < 32; i++ {
		execution, err := vm.ExecuteAsync("run")
		if err != nil {
			t.Fatalf("iteration %d start: %v", i, err)
		}
		cancelDone := make(chan error, 1)
		start := make(chan struct{})
		go func() {
			<-start
			cancelDone <- execution.Cancel()
		}()
		close(start)

		_, waitErr := execution.Wait()
		if err := <-cancelDone; err != nil {
			t.Fatalf("iteration %d cancel: %v", i, err)
		}
		var native *Error
		if !errors.As(waitErr, &native) || native.Code != ErrCodeInterrupted {
			t.Fatalf("iteration %d wait: want ErrCodeInterrupted, got %v", i, waitErr)
		}
	}
}

func TestCancellationRacePoisonsBeforeExecutionGateReopens(t *testing.T) {
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	instantiateVMFixture(t, vm, testwasm.AddModule())

	execution, err := vm.ExecuteAsync("add", I32(20), I32(22))
	if err != nil {
		t.Fatal(err)
	}
	if !execution.WaitFor(time.Second) {
		t.Fatal("finite execution did not complete")
	}
	// Model the WasmEdge 0.17.1 window in which Cancel observed an unfinished
	// async object but the worker completed before consuming the shared token.
	execution.cancelRequested.Store(true)
	if _, err := execution.Wait(); !errors.Is(err, ErrCancellationRace) {
		t.Fatalf("terminal result: got %v, want ErrCancellationRace", err)
	}
	if _, err := vm.ExecuteAsync("add", I32(1), I32(2)); !errors.Is(err, ErrUnusable) {
		t.Fatalf("next invocation: got %v, want ErrUnusable", err)
	}
}

func TestExecutionFastCompletionCancelStress(t *testing.T) {
	for i := 0; i < 48; i++ {
		vm, err := NewVM(nil)
		if err != nil {
			t.Fatal(err)
		}
		instantiateVMFixture(t, vm, testwasm.AddModule())

		execution, err := vm.ExecuteAsync("add", I32(20), I32(22))
		if err != nil {
			_ = vm.Close()
			t.Fatalf("iteration %d start: %v", i, err)
		}
		start := make(chan struct{})
		cancelDone := make(chan error, 1)
		go func() {
			<-start
			cancelDone <- execution.Cancel()
		}()
		close(start)

		var terminalErr error
		if i%2 == 0 {
			_, terminalErr = execution.Wait()
		} else {
			terminalErr = execution.Close()
		}
		cancelErr := <-cancelDone
		if cancelErr != nil && !errors.Is(cancelErr, ErrClosed) {
			_ = vm.Close()
			t.Fatalf("iteration %d cancel: %v", i, cancelErr)
		}

		var native *Error
		switch {
		case terminalErr == nil:
		case errors.Is(terminalErr, ErrCancellationRace):
			if _, err := vm.ExecuteAsync(
				"add", I32(1), I32(2),
			); !errors.Is(err, ErrUnusable) {
				_ = vm.Close()
				t.Fatalf("iteration %d post-race: got %v, want ErrUnusable", i, err)
			}
		case errors.As(terminalErr, &native) && native.Code == ErrCodeInterrupted:
		default:
			_ = vm.Close()
			t.Fatalf("iteration %d terminal: unexpected %v", i, terminalErr)
		}
		if err := vm.Close(); err != nil {
			t.Fatalf("iteration %d VM close: %v", i, err)
		}
	}
}

func TestExecutionCloseConsumesCompletedHostError(t *testing.T) {
	sentinel := errors.New("abandoned async host error")
	function := MustWrapFunc(func() error { return sentinel })
	defer function.Close()
	executor, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()

	execution, err := executor.InvokeAsync(function)
	if err != nil {
		t.Fatal(err)
	}
	if !execution.WaitFor(time.Second) {
		_ = execution.Close()
		t.Fatal("host callback did not complete")
	}
	if !hostErrorRegistryContains(sentinel) {
		_ = execution.Close()
		t.Fatal("completed callback error was not awaiting collection")
	}
	if err := execution.Close(); err != nil {
		t.Fatalf("abandon completed execution: %v", err)
	}
	if hostErrorRegistryContains(sentinel) {
		t.Fatal("Execution.Close left its host error in the registry")
	}
}

func hostErrorRegistryContains(target error) bool {
	hostErrorRegistry.Lock()
	defer hostErrorRegistry.Unlock()
	for _, entry := range hostErrorRegistry.entries {
		if entry.err == target {
			return true
		}
	}
	return false
}

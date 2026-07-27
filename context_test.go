package wasmedge

import (
	"context"
	"errors"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

type invocationContextKey struct{}

func TestExecutorInvokeContextReachesHostCallback(t *testing.T) {
	started := make(chan struct{})
	host := MustWrapFunc(func(call *CallContext, a, b int32) (int32, error) {
		if got := call.Context().Value(invocationContextKey{}); got != "executor" {
			return 0, errors.New("host callback did not receive executor context")
		}
		close(started)
		<-call.Context().Done()
		return 0, call.Context().Err()
	})
	executor, instance := instantiateWithHost(t, host)
	callHost, ok := instance.Function("call_host")
	if !ok {
		t.Fatal("call_host export missing")
	}

	base := context.WithValue(context.Background(), invocationContextKey{}, "executor")
	ctx, cancel := context.WithCancel(base)
	defer cancel()
	go func() {
		<-started
		cancel()
	}()

	if _, err := executor.InvokeContext(ctx, callHost, I32(20), I32(22)); !errors.Is(err, context.Canceled) {
		t.Fatalf("InvokeContext: want context.Canceled, got %v", err)
	}
}

func TestVMRunContextReachesHostCallback(t *testing.T) {
	started := make(chan struct{})
	env := NewModule("env")
	if err := env.AddFunction("host_add", MustWrapFunc(
		func(call *CallContext, a, b int32) (int32, error) {
			if got := call.Context().Value(invocationContextKey{}); got != "vm" {
				return 0, errors.New("host callback did not receive VM context")
			}
			close(started)
			<-call.Context().Done()
			return 0, call.Context().Err()
		},
	)); err != nil {
		t.Fatal(err)
	}

	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = vm.Close()
		_ = env.Close()
	})
	if err := vm.RegisterImport(env); err != nil {
		t.Fatal(err)
	}

	base := context.WithValue(context.Background(), invocationContextKey{}, "vm")
	ctx, cancel := context.WithCancel(base)
	defer cancel()
	go func() {
		<-started
		cancel()
	}()

	if _, err := vm.RunBytesContext(
		ctx, testwasm.HostCallModule(), "call_host", I32(20), I32(22),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("RunBytesContext: want context.Canceled, got %v", err)
	}
}

func TestVMExecuteRegisteredContextReachesHostCallback(t *testing.T) {
	started := make(chan struct{})
	env := NewModule("env")
	if err := env.AddFunction("host_add", MustWrapFunc(
		func(call *CallContext, a, b int32) (int32, error) {
			if got := call.Context().Value(invocationContextKey{}); got != "registered" {
				return 0, errors.New("host callback did not receive registered context")
			}
			close(started)
			<-call.Context().Done()
			return 0, call.Context().Err()
		},
	)); err != nil {
		t.Fatal(err)
	}

	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = vm.Close()
		_ = env.Close()
	})
	if err := vm.RegisterImport(env); err != nil {
		t.Fatal(err)
	}

	base := context.WithValue(context.Background(), invocationContextKey{}, "registered")
	ctx, cancel := context.WithCancel(base)
	defer cancel()
	go func() {
		<-started
		cancel()
	}()

	if _, err := vm.ExecuteRegisteredContext(
		ctx, "env", "host_add", I32(20), I32(22),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("ExecuteRegisteredContext: want context.Canceled, got %v", err)
	}
}

func TestCallContextBindingClearsBeforeSynchronousInvocation(t *testing.T) {
	var values []any
	var done []<-chan struct{}
	host := MustWrapFunc(func(call *CallContext, a, b int32) int32 {
		values = append(values, call.Context().Value(invocationContextKey{}))
		done = append(done, call.Context().Done())
		return a + b
	})
	executor, instance := instantiateWithHost(t, host)
	callHost, ok := instance.Function("call_host")
	if !ok {
		t.Fatal("call_host export missing")
	}
	ctx := context.WithValue(context.Background(), invocationContextKey{}, "completed")
	out, err := executor.InvokeContext(ctx, callHost, I32(20), I32(22))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].I32() != 42 {
		t.Fatalf("context host result: %v", out)
	}

	out, err = invokeCallHost(t, executor, instance, 20, 22)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].I32() != 42 {
		t.Fatalf("synchronous host result: %v", out)
	}
	if len(values) != 2 || values[0] != "completed" || values[1] != nil {
		t.Fatalf("host contexts leaked across invocations: %#v", values)
	}
	if len(done) != 2 || done[1] != nil {
		t.Fatal("synchronous host callback did not receive context.Background")
	}
}

// Package wasmedge provides Go bindings for the WasmEdge runtime C API.
//
// The package supports the active Go 1.25 and 1.26 release lines. It requires
// a WasmEdge shared library (>= 0.17.1 and < 0.18.0) at build and run time.
// Install it with the official install script, or point cgo at a local build:
//
//	export WASMEDGE_DIR=$HOME/workspace/WasmEdge
//	export CGO_CFLAGS="-I$WASMEDGE_DIR/build/include/api"
//	export CGO_LDFLAGS="-L$WASMEDGE_DIR/build/lib/api -lwasmedge"
//
// # Quick start
//
// The high-level [VM] loads, validates, instantiates and runs a module in one
// call:
//
//	vm, err := wasmedge.NewVM(nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer vm.Close()
//
//	out, err := vm.RunFile("fib.wasm", "fib", wasmedge.I32(21))
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(out[0].I32())
//
// # Explicit pipeline
//
// For embedders that need control over each stage, the [Loader], [Validator],
// [Executor] and [Store] types mirror the C API one to one:
//
//	loader, err := wasmedge.NewLoader(nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer loader.Close()
//	ast, err := loader.LoadFile("app.wasm")
//	// ... Validate, Instantiate, Invoke; see the package examples.
//
// # Host functions
//
// Go functions become WASM imports either explicitly ([NewFunction] with a
// [FunctionType]) or reflectively ([WrapFunc], which derives the WASM
// signature from the Go signature):
//
//	add, err := wasmedge.WrapFunc(func(a, b int32) int32 { return a + b })
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer add.Close()
//	mod := wasmedge.NewModule("env")
//	defer mod.Close()
//	if err := mod.AddFunction("add", add); err != nil {
//		log.Fatal(err)
//	}
//
// # WASI
//
// [NewWASIModule] returns a *[Module] and an error. Every initialization
// requires an explicit [WASIStdio] value: [DiscardWASIStdio] supplies
// non-ambient null streams, [InheritWASIStdio] grants the embedding process's
// streams, and [RedirectWASIStdio] grants three caller-owned files. An omitted
// policy or partial redirect matches [ErrInvalidArgument]. VM users normally
// enable WASI in [Config] and obtain the borrowed module with [VM.WASIModule];
// registration alone leaves its fd table empty. WASI-only methods verify
// constructor/accessor provenance rather than trusting the module name, and
// [Module.WASIExitCode] returns an error for a non-WASI receiver. Redirected
// files remain caller-owned, while the module retains their *os.File wrappers
// until module/generation teardown; discard descriptors are binding-owned.
// WasmEdge 0.17.1 fixes stdio and preopen mappings at first initialization;
// later initialization may update args and env only when those mappings are
// unchanged.
//
// # Resource management
//
// [Config], [Limits], [FunctionType], [TableType], [MemoryType], [GlobalType],
// and [TagType] are plain Go values with no Close method. Constructors that
// validate descriptors or initializers, including [NewFunction], [NewTable],
// [NewMemory], and [NewGlobal], return an error.
// [Config.Effective] resolves engine defaults and overrides into ordinary Go
// data with no native lifetime.
//
// Objects that own native WasmEdge resources implement io.Closer. Close is
// idempotent, and ownership transfers (for example adding a *[Function] to a
// *[Module]) are tracked so a transferred object is never double-freed.
// Native teardown is explicit-only: garbage collection does not replace
// Close. The wasmedge_debug build tag reports unreachable, unclosed wrappers
// but deliberately does not free their native objects.
//
// Accessors documented as "borrowed" return views bounded by their parent;
// do not use them after the parent is closed. Views into a VM's active or
// registered modules are additionally invalidated when that VM generation is
// replaced or [VM.Reset] succeeds. Reset returns an error and may report
// [ErrInUse] while an asynchronous execution or dependency lease is active.
//
// # Cancellation and asynchronous execution
//
// Context-aware methods wait for native execution to settle before returning.
// A manually managed [Execution], created by [VM.ExecuteAsync] or
// [Executor.InvokeAsync], needs exactly one terminal [Execution.Wait] or
// [Execution.Close]. [Execution.WaitFor] only observes completion, and
// [Execution.Cancel] requests interruption without releasing the execution.
// Host functions reached through a context-aware invocation receive the same
// context from [CallContext.Context] and should observe its Done channel when
// performing blocking Go work.
// If WasmEdge 0.17.1 races cancellation with native completion, the terminal
// operation reports [ErrCancellationRace] and the originating VM or Executor
// subsequently reports [ErrUnusable]; recreate it. Context methods instead
// return the context error after draining but still mark the owner unusable.
//
// # Errors
//
// Calls that can fail return error. Engine failures are *[Error] values
// carrying an [ErrCategory] and [ErrCode]; use errors.As to inspect them.
// Host functions stop execution gracefully by returning [Terminate].
// Paths, WASI/plugin/process lists, and other inputs passed to native
// NUL-terminated strings reject embedded NUL bytes with [ErrInvalidArgument].
// Driver helpers instead preserve their integer-return contract and return
// exit code 2 for an invalid argv entry.
package wasmedge

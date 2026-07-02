// Package wasmedge provides Go bindings for the WasmEdge runtime C API.
//
// The package requires a WasmEdge shared library (>= 0.17.0) at build and run
// time. Install it with the official install script, or point cgo at a local
// build:
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
//	vm, err := wasmedge.NewVM(&wasmedge.Config{WASI: true})
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
//	loader, _ := wasmedge.NewLoader(nil)
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
//	add := wasmedge.MustWrapFunc(func(a, b int32) int32 { return a + b })
//	mod, _ := wasmedge.NewModule("env")
//	mod.AddFunction("add", add)
//
// # Resource management
//
// Objects that own WasmEdge resources implement io.Closer. Close is
// idempotent, and ownership transfers (for example adding a *Function to a
// *Module) are tracked so a transferred object is never double-freed.
// Accessors documented as "borrowed" return views owned by their parent;
// do not use them after the parent is closed.
//
// # Errors
//
// Calls that can fail return error. Engine failures are *[Error] values
// carrying an [ErrCategory] and [ErrCode]; use errors.As to inspect them.
// Host functions stop execution gracefully by returning [Terminate].
package wasmedge

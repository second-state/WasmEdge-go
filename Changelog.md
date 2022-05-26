### v0.10.0 (2022-05-26)

Breaking Changes:

* `WasmEdge` updated. Please install the `WasmEdge 0.10.0` or newer version.
* The `Module`-based resource management.
  * The `Executor` will output a `Module` instance after instantiation now. Developers have the responsibility to destroy them by `(*wasmedge.Module).Release()` API.
  * Merged the `ImportObject` into the `Module`.
  * Removed the `ImportObject` structure.
* `FuncRef` mechanism changes.
  * For the better performance and security, the `FuncRef` related APIs used the `*wasmedge.Function` for the parameters and returns.
* API changes.
  * `wasmedge.NewFuncRef()` is changed to use the `*Function` as it's argument.
  * Added `(wasmedge.FuncRef).GetRef()` to retrieve the `*Function`.
  * Renamed `wasmedge.NewImportObject()` to `wasmedge.NewModule()`.
  * Renamed `(*wasmedge.ImportObject).Release()` to `(*wasmedge.Module).Release()`.
  * Renamed `(*wasmedge.ImportObject).AddFunction()` to `(*wasmedge.Module).AddFunction()`.
  * Renamed `(*wasmedge.ImportObject).AddTable()` to `(*wasmedge.Module).AddTable()`.
  * Renamed `(*wasmedge.ImportObject).AddMemory()` to `(*wasmedge.Module).AddMemory()`.
  * Renamed `(*wasmedge.ImportObject).AddGlobal()` to `(*wasmedge.Module).AddGlobal()`.
  * Renamed `(*wasmedge.ImportObject).NewWasiImportObject()` to `(*wasmedge.Module).NewWasiModule()`.
  * Renamed `(*wasmedge.ImportObject).NewWasmEdgeProcessImportObject()` to `(*wasmedge.Module).NewWasmEdgeProcessModule()`.
  * Renamed `(*wasmedge.ImportObject).InitWASI()` to `(*wasmedge.Module).InitWASI()`.
  * Renamed `(*wasmedge.ImportObject).InitWasmEdgeProcess()` to `(*wasmedge.Module).InitWasmEdgeProcess()`.
  * Renamed `(*wasmedge.ImportObject).WasiGetExitCode()` to `(*wasmedge.Module).WasiGetExitCode`.
  * Renamed `(*wasmedge.VM).RegisterImport()` to `(*wasmedge.VM).RegisterModule()`.
  * Renamed `(*wasmedge.VM).GetImportObject()` to `(*wasmedge.VM).GetImportModule()`.
  * `(*wasmedge.Store).ListFunction()` and `(*wasmedge.Store).ListFunctionRegistered()` is replaced by `(*wasmedge.Module).ListFunction()`.
  * `(*wasmedge.Store).ListTable()` and `(*wasmedge.Store).ListTableRegistered()` is replaced by `(*wasmedge.Module).ListTable()`.
  * `(*wasmedge.Store).ListMemory()` and `(*wasmedge.Store).ListMemoryRegistered()` is replaced by `(*wasmedge.Module).ListMemory()`.
  * `(*wasmedge.Store).ListGlobal()` and `(*wasmedge.Store).ListGlobalRegistered()` is replaced by `(*wasmedge.Module).ListGlobal()`.
  * `(*wasmedge.Store).FindFunction()` and `(*wasmedge.Store).FindFunctionRegistered()` is replaced by `(*wasmedge.Module).FindFunction()`.
  * `(*wasmedge.Store).FindTable()` and `(*wasmedge.Store).FindTableRegistered()` is replaced by `(*wasmedge.Module).FindTable()`.
  * `(*wasmedge.Store).FindMemory()` and `(*wasmedge.Store).FindMemoryRegistered()` is replaced by `(*wasmedge.Module).FindMemory()`.
  * `(*wasmedge.Store).FindGlobal()` and `(*wasmedge.Store).FindGlobalRegistered()` is replaced by `(*wasmedge.Module).FindGlobal()`.
* Temporarily removed the `wasmedge_process` related APIs for supporting plug-in architecture in the future.
  * Removed the `(*wasmedge.Module).NewWasmEdgeProcessModule()` API.
  * Removed the `(*wasmedge.Module).InitWasmEdgeProcess()` API.

Features:

* Updated to the [WasmEdge 0.10.0](https://github.com/WasmEdge/WasmEdge/releases/tag/0.10.0).

Documentation:

* Updated the [documentation](https://wasmedge.org/book/en/embed/go/ref.html).

### v0.9.2 (2022-02-11)

This version is the bug fixing for `WasmEdge-go v0.9.1`, and the version `v0.9.1` is retracted.
Developers should install the [WasmEdge 0.9.1](https://github.com/WasmEdge/WasmEdge/releases/tag/0.9.1) for using this package.

Fixed issues:

* Fixed the lost headers.

Features:

* Updated to the [WasmEdge 0.9.1](https://github.com/WasmEdge/WasmEdge/releases/tag/0.9.1).
* Added the new APIs.
  * Added the `Async` struct for asynchronize function execution.
    * Added `(*Async).WaitFor` API.
    * Added `(*Async).Cancel` API.
    * Added `(*Async).GetResult` API.
    * Added `(*Async).Release` API.
  * Added the asynchronize function execution in `VM`.
    * Added `(*VM).AsyncRunWasmFile` API.
    * Added `(*VM).AsyncRunWasmBuffer` API.
    * Added `(*VM).AsyncRunWasmAST` API.
    * Added `(*VM).AsyncExecute` API.
    * Added `(*VM).AsyncExecuteRegistered` API.
* Synchronized and Updated the `Proposal` order with `WasmEdge 0.9.1`.

### v0.9.1 (2022-02-10) (Retract)

Features:

* Updated to the [WasmEdge 0.9.1](https://github.com/WasmEdge/WasmEdge/releases/tag/0.9.1).
* Added the new APIs.
  * Added the `Async` struct for asynchronize function execution.
    * Added `(*Async).WaitFor` API.
    * Added `(*Async).Cancel` API.
    * Added `(*Async).GetResult` API.
    * Added `(*Async).Release` API.
  * Added the asynchronize function execution in `VM`.
    * Added `(*VM).AsyncRunWasmFile` API.
    * Added `(*VM).AsyncRunWasmBuffer` API.
    * Added `(*VM).AsyncRunWasmAST` API.
    * Added `(*VM).AsyncExecute` API.
    * Added `(*VM).AsyncExecuteRegistered` API.
* Synchronized and Updated the `Proposal` order with `WasmEdge 0.9.1`.

### v0.9.0 (2021-12-09)

Breaking Changes:

* `WasmEdge` updated. Please install the `WasmEdge 0.9.0` or newer version.
* Resource releasing behavior changes.
  * Renamed the `Delete` functions into `Release`.
  * Developers should call the `Release` methods of objects that created by themselves to release resources.
* API changes.
  * AST
    * Renamed `(*AST).Delete` to `(*AST).Release`.
  * Limit
    * Removed `(*Limit).WithMaxVal`.
  * Configure
    * Renamed `(*Configure).Delete` to `(*Configure).Release`.
    * Renamed `(*Configure).SetCompilerInstructionCounting` to `(*Configure).SetStatisticsInstructionCounting`.
    * Renamed `(*Configure).IsCompilerInstructionCounting` to `(*Configure).IsStatisticsInstructionCounting`.
    * Renamed `(*Configure).SetCompilerCostMeasuring` to `(*Configure).SetStatisticsCostMeasuring`.
    * Renamed `(*Configure).IsCompilerCostMeasuring` to `(*Configure).IsStatisticsCostMeasuring`.
  * Statistics
    * Renamed `(*Statistics).Delete` to `(*Statistics).Release`.
  * Compiler
    * Renamed `(*Compiler).Delete` to `(*Compiler).Release`.
  * Loader
    * Renamed `(*Loader).Delete` to `(*Loader).Release`.
  * Validator
    * Renamed `(*Validator).Delete` to `(*Validator).Release`.
  * Interpreter: Renamed `Interpreter` to `Executor`
    * Renamed `NewInterpreter` to `NewExecutor`.
    * Renamed `NewInterpreterWithConfig` to `NewExecutorWithConfig`.
    * Renamed `NewInterpreterWithStatistics` to `NewExecutorWithStatistics`.
    * Renamed `NewInterpreterWithConfigAndStatistics` to `NewExecutorWithConfigAndStatistics`.
    * Renamed `(*Interpreter).Instantiate` to `(*Executor).Instantiate`.
    * Renamed `(*Interpreter).RegisterImport` to `(*Executor).RegisterImport`.
    * Renamed `(*Interpreter).RegisterModule` to `(*Executor).RegisterModule`.
    * Renamed `(*Interpreter).Invoke` to `(*Executor).Invoke`.
    * Renamed `(*Interpreter).InvokeRegistered` to `(*Executor).InvokeRegistered`.
    * Renamed `(*Interpreter).Delete` to `(*Executor).Release`.
  * Store
    * Renamed `(*Store).Delete` to `(*Store).Release`.
  * ImportObject
    * Removed the `additional` column in `NewImportObject`. The additional data to set into host functions are in the `NewFunction` now.
    * Removed the `dirs` column in `NewWasiImportObject` and `InitWasi`. Please combine the `dirs` list into the `preopens`.
    * Renamed `(*ImportObject).Delete` to `(*ImportObject).Release`.
    * Renamed `(*ImportObject).AddHostFunction` to `(*ImportObject).AddFunction`.
  * Instances
    * Merged `HostFunction` into `Function`.
    * Renamed `NewHostFunction` to `NewFunction`.
    * Renamed `(*HostFunction).Delete` to `(*Function).Release`.
    * Added the `additional` column in `NewFunction`.
    * Modified the `NewTable` API.
    * Renamed `(*Table).Delete` to `(*Table).Release`.
    * Modified the `NewMemory` API.
    * Renamed `(*Memory).Delete` to `(*Memory).Release`.
    * Modified the `NewGlobal` API.
    * Renamed `(*Global).Delete` to `(*Global).Release`.

Features:

* Updated to the [WasmEdge 0.9.0](https://github.com/WasmEdge/WasmEdge/releases/tag/0.9.0).
  * Added the new APIs.
* Added the CI for testing with [WasmEdge-go-examples](https://github.com/second-state/WasmEdge-go-examples/).

Fixed issues:

* Fixed the bugs in the load-WASM-from-buffer functions.
* Fixed the bugs in bindgen execution functions.
* Fixed the memory issue in `(*Memory).GetData`. Wrap the memory instead of copying.

Documentation:

* Updated the installation guide.
* Added the [quick start guide](https://github.com/second-state/WasmEdge-go/blob/master/docs/go_api.md).

### v0.8.2 (2021-09-09)

Features:

* Updated to the [WasmEdge 0.8.2](https://github.com/WasmEdge/WasmEdge/releases/tag/0.8.2).
* Added the CI for build testing with every tags.

Fixed issues:

* Fixed the GC slice in host function implementation.

Docmentation:

* Added the golang version notification.
* Added the example links.

### v0.8.1 (2021-06-25)

Features:

* Updated to the [WasmEdge 0.8.1](https://github.com/WasmEdge/WasmEdge/releases/tag/0.8.1).
* WasmEdge Golang binding for C API
  * Added the new APIs about compiler options.
  * Added the new APIs about `wasmedge_process` settings.

### v0.8.0 (2021-06-01)

Features:

* WasmEdge Golang binding for C API
  * Please refer to the [README](https://github.com/second-state/WasmEdge-go/blob/master/README.md) for installation.
  * Update to the [WasmEdge 0.8.0](https://github.com/WasmEdge/WasmEdge/releases/tag/0.8.0).
* WasmEdge-go for tensorflow extension
  * The extension of [WasmEdge-tensorflow](https://github.com/second-state/WasmEdge-tensorflow) for supplying the tensorflow related host functions.
  * Please refer to the [MTCNN](https://github.com/second-state/WasmEdge-go-examples/tree/master/go_mtcnn) example.
* WasmEdge-go for image extension
  * The extension of [WasmEdge-image](https://github.com/second-state/WasmEdge-image) for supplying the image related host functions.
* Wasm-bindgen for Golang
  * Support Wasm-bindgen in WasmEdge-go.
  * Please refer to the [BindgenFuncs](https://github.com/second-state/WasmEdge-go-examples/tree/master/go_BindgenFuncs) example.

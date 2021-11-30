### v0.9.0-rc5 (2021-11-30)

Breaking Changes:

* `WasmEdge` updated. Please install the `WasmEdge 0.9.0-rc.5` or newer version.
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

* Updated to the [WasmEdge 0.9.0-rc.5](https://github.com/WasmEdge/WasmEdge/releases/tag/0.9.0-rc.5).
  * Added the new APIs.
* Added the CI for testing with [WasmEdge-go-examples](https://github.com/second-state/WasmEdge-go-examples/).

Fixed issues:

* Fixed the bugs in the load-WASM-from-buffer functions.
* Fixed the bugs in bindgen execution functions.
* Fixed the memory issue in `(*Memory).GetData`. Wrap the memory instead of copying.

Documentation:

* Updated the installation guide.
* Added the [quick start guide](https://github.com/second-state/WasmEdge-go/blob/master/docs/go_api.md).

### v0.9.0-rc4 (2021-11-23)

Breaking Changes:

* `WasmEdge` updated. Please install the `WasmEdge 0.9.0-rc.4` or newer version.
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

* Updated to the [WasmEdge 0.9.0-rc.4](https://github.com/WasmEdge/WasmEdge/releases/tag/0.9.0-rc.4).
  * Added the new APIs.
* Added the CI for testing with [WasmEdge-go-examples](https://github.com/second-state/WasmEdge-go-examples/).

Fixed issues:

* Fixed the bugs in the load-WASM-from-buffer functions.
* Fixed the bugs in bindgen execution functions.
* Fixed the memory issue in `(*Memory).GetData`. Wrap the memory instead of copying.

Documentation:

* Updated the installation guide.
* Added the [quick start guide](https://github.com/second-state/WasmEdge-go/blob/master/docs/go_api.md).

### v0.9.0-rc3 (2021-11-03)

Breaking Changes:

* `WasmEdge` updated. Please install the `WasmEdge 0.9.0-rc.2` or newer version.
* Behavior changes.
  * Added the finalizer mechanism in all objects.
  * Developers can call the `Release` methods of objects to forcely release resources.
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

* Updated to the [WasmEdge 0.9.0-rc.2](https://github.com/WasmEdge/WasmEdge/releases/tag/0.9.0-rc.2).
  * Added the new APIs.
* Added the CI for testing with [WasmEdge-go-examples](https://github.com/second-state/WasmEdge-go-examples/).

Fixed issues:

* Fixed the bugs in the load-WASM-from-buffer functions.
* Fixed the bugs in bindgen execution functions.

Documentation:

* Updated the installation guide.
* Added the [quick start guide](https://github.com/second-state/WasmEdge-go/blob/master/docs/go_api.md).

### v0.9.0-rc2 (2021-10-23)

Documentation:

* Updated the installation guide.

### v0.9.0-rc1 (2021-09-24)

Fixed issues:

* Fixed the bugs in the load-WASM-from-buffer functions.
* Fixed the bugs in bindgen execution functions.

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

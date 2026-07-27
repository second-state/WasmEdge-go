# Migrating from WasmEdge-go v1 to v2

v2 is a ground-up redesign targeting WasmEdge ≥ 0.17.1 and < 0.18.0, with Go
1.25 and 1.26 as the active supported toolchains. The import path changes and
every symbol was reviewed; this table maps the common surface.

## Import path

```go
// v1
import "github.com/second-state/WasmEdge-go/wasmedge"
// v2
import wasmedge "github.com/second-state/WasmEdge-go/v2"
```

## Concepts

| v1 | v2 |
|---|---|
| `Result`, `res.GetMessage()` | `error`; inspect with `var we *wasmedge.Error; errors.As(err, &we)` |
| `obj.Release()` | `obj.Close()` (idempotent, `io.Closer`; explicit Close is required) |
| `interface{}` values | `wasmedge.Value` — `I32(7)`, `v.I32()`, `V128([16]byte{…})` |
| `NewConfigure(wasmedge.WASI)` + `Release` | `&wasmedge.Config{WASI: true}` (plain data, no Close), then explicitly initialize the borrowed WASI module with a `WASIStdio` policy |
| mutating/querying a native Configure object | declarative `Config`; call `Config.Effective()` for the complete native defaults plus overrides |
| `NewExternRef(&obj)` + registry | `NewExternRef(obj)` + `ExternRefValue`; close the owning `ExternRef` explicitly |
| heap-backed type objects + `Release` | plain `FunctionType`, `TableType`, `MemoryType`, `GlobalType`, and `TagType` values; no Close |
| host func `func(data interface{}, cf *CallingFrame, params []interface{}) ([]interface{}, Result)` | `HostFunc: func(*CallContext, []Value) ([]Value, error)`, or just `WrapFunc(func(a, b int32) int32 {…})` |
| build tags `tensorflow`, `image` | removed — load plugins instead (`LoadPluginsFromDefaultPaths`) |

## VM

| v1 | v2 |
|---|---|
| `NewVM()` / `NewVMWithConfig(conf)` | `NewVM(nil)` / `NewVM(cfg)` |
| `NewVMWithStore(s)` / `...ConfigAndStore` | `NewVM(cfg, wasmedge.WithExternalStore(s))` |
| `vm.RunWasmBuffer(buf, fn, args...)` | `vm.RunBytes(b, fn, params...)` |
| `vm.RunWasmFile(path, fn, args...)` | `vm.RunFile(...)` |
| `vm.LoadWasmBuffer/File/AST` | `vm.LoadBytes/LoadFile/Load` |
| `vm.Validate()` / `vm.Instantiate()` | same names, `error` results |
| `vm.Execute(fn, args...)` | `vm.Execute(fn, params...)` |
| `vm.ExecuteRegistered(mod, fn, ...)` | `vm.ExecuteRegistered(...)` |
| `vm.RegisterWasmBuffer(name, buf)` | `vm.RegisterModuleBytes(name, b)` |
| `vm.RegisterImport(imp)` | `vm.RegisterImport(mod)` |
| `vm.GetFunctionList()` | `vm.Functions()` (`[]VMFunction`) |
| `vm.GetFunctionType(fn)` | `vm.FunctionType(fn)` (`, ok` form) |
| `vm.GetActiveModule()` | `vm.ActiveModule()` |
| `vm.GetStore()` etc. | `vm.Store()`, `vm.Stats()` (borrowed) |
| `vm.Cleanup()` | `vm.Reset() error`; invalidates active and registered borrowed views |
| — | `vm.ExecuteContext(ctx, fn, …)`, `vm.ExecuteAsync(fn, …) (*Execution, error)` |

## Pipeline

| v1 | v2 |
|---|---|
| `NewLoader()` / `NewLoaderWithConfig` | `NewLoader(nil)` / `NewLoader(cfg)` |
| `loader.LoadBuffer(buf)` | `loader.LoadBytes(b)` |
| `validator.Validate(ast)` | same, `error` result |
| `NewExecutor()` / `...WithStatistics` | `NewExecutor(nil)` / `NewExecutor(cfg, WithStats(s))` |
| `executor.Instantiate(store, ast)` | same shape, returns `(*Module, error)` |
| `executor.Invoke(fn, params...)` | same shape, typed values |
| — | `executor.InvokeAsync(fn, …)`, `executor.InvokeContext(ctx, fn, …)`, `RegisterImportWithAlias` |

`WithExternalStore` and `WithStats` lease their caller-owned dependencies.
Close the VM/Executor first; an early dependency `Close` returns `ErrInUse`.
`Statistics.SetCostTable` now returns `error` so an unrepresentable native
table length is reported rather than narrowed at the ABI boundary.

## Instances

| v1 | v2 |
|---|---|
| `NewModule(name)` | `NewModule(name)` (+ `NewModuleWithData`) |
| `mod.AddFunction/Table/Memory/Global` | same names; ownership transfer is tracked |
| `NewFunctionType(params, returns)` | `FunctionType{Params: params, Results: returns}` |
| `NewMemoryType(limit)` | `MemoryType{Limits: limit}` |
| `NewTableType(refType, limit)` | `TableType{Element: refType, Limits: limit}` |
| `NewGlobalType(valueType, mutability)` | `GlobalType{Value: valueType, Mutability: mutability}` |
| one-result `NewFunction/Table/Memory/Global` | descriptor-taking constructors returning `(*T, error)` |
| `mod.FindFunction(name)` | `mod.Function(name)` (`, ok` form; borrowed) |
| `mem.GetData(off, len)` | `mem.Read(off, len)` |
| `mem.SetData(data, off)` | `mem.Write(off, data)` |
| `mem.GetPointer(...)` | `mem.UnsafeSlice(off, len)` (documented lifetime) |
| `mem.GetPageSize()` / `GrowPage(n)` | `mem.PageCount()` / `GrowPages(n)` |
| `NewMemory(NewMemoryType(NewLimit(1)))` | `NewMemory(MemoryType{Limits: Limits{Min: 1}})` returning `(*Memory, error)` |
| `tab.GetData/SetData` | `tab.Get/Set` |
| `glob.GetValue/SetValue` | `glob.Value/SetValue` |

Reference-valued table/global initializers and setters require the exact
declared `ValType`; null is rejected for a non-nullable reference type before
native state is mutated.

## WASI, plugins, logging

| v1 | v2 |
|---|---|
| `NewWasiImportObject(args, envs, preopens)` | `NewWASIModule(WASIConfig{…, Stdio: InheritWASIStdio()})` returning `(*Module, error)` |
| `vm.GetImportModule(wasmedge.WASI)` | `vm.WASIModule()` |
| `wasi.WasiGetExitCode()` | `code, err := mod.WASIExitCode()`; `err` is `ErrNotWASIModule` without trusted WASI provenance |
| `LoadPluginDefaultPaths()` | `LoadPluginsFromDefaultPaths()` |
| plugin/process initialization with no status | `LoadPlugins`, `InitWASINN`, and `InitWasmEdgeProcess` return `error` |
| `SetLogErrorLevel()` / `SetLogOff()` | `SetLogLevel(LogLevelError)` / `SetLogOff()` |
| — | `SetLogCallback(cb)`, `RouteLogsToSlog(logger)` |

v1 implicitly inherited process stdio. v2 does not: registering the VM's WASI
module leaves its fd table empty, and every `NewWASIModule`/`InitWASI`
initialization requires `DiscardWASIStdio()`, `InheritWASIStdio()`, or
`RedirectWASIStdio(in, out, errOut)`. Use discard for stdin EOF plus output
sinks without ambient host streams. Omitting the choice returns
`ErrWASIStdioPolicyRequired`, matching `ErrInvalidArgument`.

Redirected files stay caller-owned. Keep them open while the configuration is
active; the module keeps their Go wrappers reachable until
module/VM-generation teardown. WasmEdge 0.17.1 cannot replace entries in an
initialized WASI fd table, so later `InitWASI` calls may change `Args` and
`Envs` but must repeat the exact same `Preopens`, stdio policy, and files. A
mapping change returns `ErrWASIResourceMappingImmutable` (wrapping
`errors.ErrUnsupported`) before native state changes. On Unix, each redirected
descriptor is also checked to be open and representable as an `int32` before
native initialization.
Strings passed to native C-string APIs reject embedded NUL bytes with
`ErrInvalidArgument`; Driver helpers return exit code 2 for invalid argv.

## Lifetime differences

Garbage collection does not destroy native WasmEdge objects in v2. Every
owned wrapper must be closed explicitly; `-tags wasmedge_debug` only reports
a wrapper that became unreachable without `Close`.

Borrowed views remain bounded by their owner. VM active-module views also
belong to a generation: a successful `Instantiate`, a started one-shot
`Run*`, or `Reset` can invalidate an older view even though the VM itself
remains open.
`Reset` returns an error (notably `ErrInUse` while an asynchronous execution
or dependency lease is active).

For manual asynchronous execution, call exactly one of `Execution.Wait` or
`Execution.Close`. `WaitFor` does not consume the execution, and `Cancel`
does not release it. If WasmEdge 0.17.1 races cancellation with native
completion, the manual terminal call returns `ErrCancellationRace` and the
originating VM/Executor becomes `ErrUnusable`; context methods return the
context error but impose the same recreation requirement.

Host functions invoked by `InvokeContext`, `ExecuteContext`, or `RunContext`
receive the same context from `CallContext.Context()`. Synchronous and
manually managed asynchronous calls receive `context.Background()`.

## Known 0.17.1 constraints

- The official `darwin/arm64` SDK can abort the process in
  `Loader.Serialize`. v2 returns `ErrSerializeUnsupported` (wrapping
  `errors.ErrUnsupported`) before native entry on that platform; other
  platforms keep native serialization enabled.
- The official SDK does not export the build-conditional WASI-NN RPC driver.
  The complete declaration accounting and future enablement requirements are
  in [API_COVERAGE.md](API_COVERAGE.md).

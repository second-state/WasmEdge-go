# Migrating from WasmEdge-go v1 to v2

v2 is a ground-up redesign targeting WasmEdge ≥ 0.17. The import path
changes and every symbol was reviewed; this table maps the common surface.
Where a v2 cell says *(A#)* the convenience is an open starter task in
PLAN.md Appendix A — the underlying capability already exists through the
explicit form.

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
| `Result`, `res.GetMessage()` | `error`; inspect with `errors.As(err, &*wasmedge.Error)` |
| `obj.Release()` | `obj.Close()` (idempotent, `io.Closer`) |
| `interface{}` values | `wasmedge.Value` — `I32(7)`, `v.I32()`, `V128([16]byte{…})` |
| `NewConfigure(wasmedge.WASI)` + `Release` | `&wasmedge.Config{WASI: true}` (plain data, no Close) |
| `NewExternRef(&obj)` + registry | `NewExternRef(obj)` (`cgo.Handle`-backed) + `ExternRefValue` |
| host func `func(data interface{}, cf *CallingFrame, params []interface{}) ([]interface{}, Result)` | `HostFunc: func(*CallContext, []Value) ([]Value, error)`, or just `WrapFunc(func(a, b int32) int32 {…})` |
| build tags `tensorflow`, `image` | removed — load plugins instead (`LoadPluginsFromDefaultPaths`) |

## VM

| v1 | v2 |
|---|---|
| `NewVM()` / `NewVMWithConfig(conf)` | `NewVM(nil)` / `NewVM(cfg)` |
| `NewVMWithStore(s)` / `...ConfigAndStore` | `NewVM(cfg, wasmedge.WithExternalStore(s))` |
| `vm.RunWasmBuffer(buf, fn, args...)` | `vm.RunBytes(b, fn, params...)` |
| `vm.RunWasmFile(path, fn, args...)` | `vm.RunFile(...)` *(A4)*; staged `LoadFile`+… today |
| `vm.LoadWasmBuffer/File/AST` | `vm.LoadBytes/LoadFile/Load` |
| `vm.Validate()` / `vm.Instantiate()` | same names, `error` results |
| `vm.Execute(fn, args...)` | `vm.Execute(fn, params...)` |
| `vm.ExecuteRegistered(mod, fn, ...)` | `vm.ExecuteRegistered(...)` *(A4)* |
| `vm.RegisterWasmBuffer(name, buf)` | `vm.RegisterModuleBytes(name, b)` |
| `vm.RegisterImport(imp)` | `vm.RegisterImport(mod)` |
| `vm.GetFunctionList()` | `vm.Functions()` (`[]VMFunction`) |
| `vm.GetFunctionType(fn)` | `vm.FunctionType(fn)` (`, ok` form) |
| `vm.GetActiveModule()` | `vm.ActiveModule()` |
| `vm.GetStore()` etc. | `vm.Store()`, `vm.Stats()` (borrowed) |
| `vm.Cleanup()` | `vm.Reset()` |
| — | `vm.ExecuteContext(ctx, fn, …)`, `vm.ExecuteAsync(fn, …)` |

## Pipeline

| v1 | v2 |
|---|---|
| `NewLoader()` / `NewLoaderWithConfig` | `NewLoader(nil)` / `NewLoader(cfg)` |
| `loader.LoadBuffer(buf)` | `loader.LoadBytes(b)` |
| `validator.Validate(ast)` | same, `error` result |
| `NewExecutor()` / `...WithStatistics` | `NewExecutor(nil)` / `NewExecutor(cfg, WithStats(s))` |
| `executor.Instantiate(store, ast)` | same shape, returns `(*Module, error)` |
| `executor.Invoke(fn, params...)` | same shape, typed values |
| — | `executor.InvokeContext(ctx, fn, …)`, `RegisterImportWithAlias` |

## Instances

| v1 | v2 |
|---|---|
| `NewModule(name)` | `NewModule(name)` (+ `NewModuleWithData`) |
| `mod.AddFunction/Table/Memory/Global` | same names; ownership transfer is tracked |
| `mod.FindFunction(name)` | `mod.Function(name)` (`, ok` form; borrowed) |
| `mem.GetData(off, len)` | `mem.Read(off, len)` |
| `mem.SetData(data, off)` | `mem.Write(off, data)` |
| `mem.GetPointer(...)` | `mem.UnsafeSlice(off, len)` (documented lifetime) |
| `mem.GetPageSize()` / `GrowPage(n)` | `mem.PageCount()` / `GrowPages(n)` |
| `NewMemory(NewMemoryType(NewLimit(1)))` | `NewMemory(NewMemoryType(Limits{Min: 1}))` |
| `tab.GetData/SetData` | `tab.Get/Set` |
| `glob.GetValue/SetValue` | `glob.Value/SetValue` *(A7)* |

## WASI, plugins, logging

| v1 | v2 |
|---|---|
| `NewWasiImportObject(args, envs, preopens)` | `NewWASIModule(WASIConfig{…})` |
| `vm.GetImportModule(wasmedge.WASI)` | `vm.WASIModule()` |
| `wasi.WasiGetExitCode()` | `mod.WASIExitCode()` |
| `LoadPluginDefaultPaths()` | `LoadPluginsFromDefaultPaths()` |
| `SetLogErrorLevel()` / `SetLogOff()` | `SetLogLevel(LogLevelError)` / `SetLogOff()` |
| — | `SetLogCallback(cb)`, `RouteLogsToSlog(logger)` |

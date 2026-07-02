# WasmEdge-go v2 Redesign Specification

Status: PROPOSED (awaiting maintainer review)
Author: hydai (drafted with Claude Code)
Date: 2026-07-02
Target: WasmEdge C API >= 0.17.0

## 1. Motivation

WasmEdge-go was written against the 0.9.x C API and grew organically to 0.14.
It carries patterns that predate modern Go and modern WasmEdge:

- **Untyped values.** Every WASM value crosses the API as `interface{}`.
  Type errors surface at runtime as panics deep inside the binding.
- **Manual memory management.** 20 types expose `Release()`; forgetting one
  leaks C memory, calling one twice corrupts the heap. There is no safety net
  and no idempotency guarantee.
- **Non-Go error handling.** Functions return a `Result` struct that callers
  must interrogate with `.GetMessage()`; nothing satisfies the `error`
  interface, so `errors.Is`/`errors.As` and every Go error idiom are useless.
- **Deprecated runtime machinery.** `reflect.SliceHeader` (deprecated since
  Go 1.20), `panic()` inside cgo callbacks (undefined behavior when it unwinds
  into C), a hand-rolled mutex-protected host-function registry predating
  `runtime/cgo.Handle`.
- **Stale API surface.** The 0.17 C API restructured headers, made `ValType`
  a struct, turned `Limit` into a context with Memory64/Threads flags, added
  Tags (exception handling), GC types, `Standard`/`RunMode` configuration,
  log callbacks, and explicitly deprecated the buffer-based entry points the
  current binding still uses.
- **Legacy extensions.** TensorFlow/Image-era build tags and host
  registrations no longer exist upstream; plugins are the only extension
  mechanism now.

Since breaking changes are allowed, we redesign the public API instead of
patching it.

## 2. Goals / Non-Goals

**Goals**

1. An idiomatic Go API: `error` returns, `Close() error`, typed values,
   `context.Context` cancellation, `log/slog` integration, doc comments and
   runnable examples on pkg.go.dev.
2. Full coverage of the **stable** 0.17 C API surface
   (`wasmedge_basic/value/configure/ast/instance/execution/vm/compiler/plugin/tools`).
3. Memory safety by construction: idempotent `Close`, ownership tracking for
   caller-owned vs. borrowed vs. transferred objects, no way to double-free
   from safe Go code, panic containment at every cgo callback boundary.
4. A scaffolding-complete codebase where every remaining gap is a documented,
   self-contained `TODO(intern-*)` task.

**Non-Goals**

- Binding `wasmedge_deprecated.h` (buffer-based calls, `ForceInterpreter`).
  v2 starts clean.
- Binding `wasmedge_experimental.h` (pre/post host-function hooks). Revisit
  when upstream stabilizes it; the design reserves a build tag for it.
- Component Model runtime support. The C API only exposes the
  `WasmEdge_ComponentTypeCode` enum today; we mirror the enum and stop there.
- Supporting WasmEdge < 0.17. One binding version tracks one C API line;
  this is the same policy the project already used (binding 0.14.x ↔ runtime
  0.14.x).
- A pure-Go execution fallback (that is wazero's niche, not ours).

## 3. Approaches Considered

**A. Modernized 1:1 mapping.** Keep the current shape (one Go func per C
func, `Release()`, `Result`) and only fix bugs/naming.
*Rejected:* perpetuates C ergonomics in Go; every caller keeps writing
resource-leak-prone code; does not exploit anything Go gained since 2021.

**B. Idiomatic two-layer binding (chosen).** Keep the C API's proven
two-layer split — a high-level `VM` for the common path and explicit
`Loader/Validator/Executor/Store` for advanced embedding — but express both
layers in idiomatic Go: typed values, `error`, `Close`, declarative `Config`,
reflection-wrapped host functions, `context.Context` async.
*Chosen:* preserves a 1:1 mental mapping to the C API docs (crucial for a
binding) while making misuse hard. This mirrors the ownership taxonomy of
the sibling Python binding redesign (`bindings/python/DESIGN.md` in the
WasmEdge repo, branch `hyda/new_python_binding`) and the Rust split
(`wasmedge-sys`/`wasmedge-sdk`); keep the three bindings' lifetime models
aligned when evolving any of them.

**C. Facade with hidden engine (wazero-style interfaces).** Define pure-Go
interfaces (`Runtime`, `CompiledModule`, …) and hide cgo entirely as a
swappable backend.
*Rejected:* heavy abstraction for a single backend; obscures the C API docs;
complicates debugging and plugin interop. YAGNI.

## 4. Module Layout

```
github.com/second-state/WasmEdge-go/v2      ← module path (Go v2 rule)
├── go.mod                                  go 1.24
├── *.go                                    package wasmedge (root package)
├── internal/testwasm/                      hand-encoded WASM fixtures (pure Go)
├── examples/                               go:build ignore examples + example_test.go
├── docs/                                   this spec, migration guide
└── .github/workflows/                      CI
```

Decisions:

- **Root package.** Import `github.com/second-state/WasmEdge-go/v2`, package
  name `wasmedge`. One import line, wazero-style. The legacy `wasmedge/`
  directory is deleted on this branch (v1 stays reachable via module
  versioning — that is exactly what `/v2` semantics are for).
- **Single cgo package.** cgo types cannot cross package boundaries, so all
  cgo lives in the root package. `internal/` holds only pure-Go helpers.
- **File-per-concept**, mirroring the upstream header split so a reader can
  diff `function.go` against `wasmedge_instance.h` section by section.

## 5. Core Design

### 5.1 Errors (`errors.go`)

```go
// Error is a WasmEdge failure. It satisfies the error interface.
type Error struct {
    Category ErrCategory // ErrCategoryWASM or ErrCategoryUserLevel
    Code     ErrCode     // 24-bit code, e.g. ErrCodeCostLimitExceeded
    Message  string      // engine message, captured at creation
}

type ErrCode uint32     // constants generated from enum.inc
type ErrCategory uint32
```

- Every C call returning `WasmEdge_Result` maps to `error` (nil on success).
- `WasmEdge_Result_Terminate` is a *successful* outcome in the C API
  (`ResultOK` returns true). We mirror that: graceful termination (e.g. WASI
  `proc_exit`) returns nil; the exit code is read from the WASI module.
- Host functions return `error`. Returning nil means success; returning
  `wasmedge.Terminate` (a sentinel) stops execution gracefully; returning a
  `*wasmedge.Error` propagates its code; any other error becomes
  `ErrCategoryUserLevel` with a generic code.
- `errors.Is` works on sentinel comparisons via `(*Error).Is`; predicate
  helpers (`IsTrap(err)`, `IsCostLimitExceeded(err)`) cover common checks.

### 5.2 Values (`value.go`, `valtype.go`)

```go
// Value is one WASM value (i32/i64/f32/f64/v128/funcref/externref).
type Value struct{ raw C.WasmEdge_Value } // opaque to users

func I32(v int32) Value
func I64(v int64) Value
func F32(v float32) Value
func F64(v float64) Value
func V128(v [16]byte) Value
func FuncRefValue(f *Function) Value
func ExternRefValue(r *ExternRef) Value

func (v Value) Kind() ValKind      // enum-like introspection
func (v Value) I32() int32         // panics on kind mismatch (reflect-style)
func (v Value) V128() [16]byte
func (v Value) IsNullRef() bool
...
```

- `ValType` wraps the 0.17 struct-based `WasmEdge_ValType` (it is no longer
  an enum upstream). Constructors `ValTypeI32()` …; predicates `IsRef()`,
  `IsRefNull()` for GC-proposal types.
- `ExternRef` pins a Go value across the boundary using `runtime/cgo.Handle`.
  It is an owned resource: `NewExternRef(v any) *ExternRef`, `Value() any`,
  `Close() error`. This replaces the old hand-rolled registry with `_valid`
  flags.
- V128 is `[16]byte` (little-endian lane order), not `big.Int`.

### 5.3 Ownership and `Close` (`resource.go`)

Every wrapper embeds a small `resource` helper that records:

- **owned** — created by us, `Close` calls the C destructor. Idempotent
  (second `Close` is a no-op returning nil). All owned types implement
  `io.Closer`.
- **borrowed** — obtained from a container (e.g. `Store.Module`,
  `Module.Function`, `CallContext.Memory`). `Close` is a no-op; validity is
  bounded by the owner and documented on the accessor. Borrowed views hold
  a strong reference to their owning wrapper, so a live view keeps the
  owner's GC safety net from freeing the underlying C object (views scoped
  to a host call are instead guarded by the `CallContext` validity flag).
- **transferred** — ownership moved into the engine (e.g. a `*Function`
  added to a `*Module` via `AddFunction`, a `Limits`/`FunctionType` consumed
  by a constructor). The transfer API disarms the wrapper so a later `Close`
  cannot double-free.

Safety net: owned objects register a `runtime.AddCleanup` (Go 1.24) that
frees the C object if the wrapper is garbage-collected without `Close`, and
reports the leak when built with `-tags wasmedge_debug`. Cleanup is disarmed
on `Close`/transfer. Explicit `Close` remains the documented contract;
the cleanup exists so a forgotten handle degrades to "eventually freed"
instead of "leaked forever".

### 5.4 Configuration (`config.go`)

The C `ConfigureContext` is only ever *read at construction time* by
VM/Loader/Validator/Executor/Compiler (they copy it). Therefore v2 makes
`Config` a plain Go struct with no C lifetime at all — one entire resource
type disappears:

```go
type Config struct {
    Standard         Standard   // 0 ⇒ engine default (WASM 2)
    EnableProposals  []Proposal // applied after Standard
    DisableProposals []Proposal
    WASI             bool       // built-in host registration
    RunMode          RunMode    // 0 = Interpreter (engine default)
    MaxMemoryPages   uint64     // 0 ⇒ engine default
    AllowAFUNIX      bool
    Compiler         CompilerConfig // OptLevel, OutputFormat, DumpIR, GenericBinary, Interruptible
    Stats            StatsConfig    // InstructionCounting, CostMeasuring, TimeMeasuring
}
```

Constructors accept `*Config` where nil means defaults:
`NewVM(nil)`, `NewVM(&Config{WASI: true, RunMode: RunModeJIT})`.
Internally `(*Config).build()` materializes a `WasmEdge_ConfigureContext`,
the constructor consumes it, and it is deleted before returning.

New 0.17 knobs (`Standard` WASM_1/2/3, `RunMode` Interpreter/JIT/AOT/LazyJIT,
`AllowAFUNIX`) are first-class fields.

### 5.5 Host Functions (`function.go`, `callcontext.go`)

Two levels:

```go
// Level 1: explicit — full control, mirrors the C API.
type HostFunc func(call *CallContext, params []Value) ([]Value, error)
func NewFunction(ft *FunctionType, fn HostFunc, opts ...FunctionOption) *Function

// Level 2: reflective — the common path. Derives the FunctionType from the
// Go signature. Supported: int32/uint32/int64/uint64/float32/float64,
// optional leading *CallContext, optional trailing error.
func WrapFunc(fn any) (*Function, error)
func MustWrapFunc(fn any) *Function

// Inside a host function:
type CallContext struct{ ... } // wraps WasmEdge_CallingFrameContext
func (c *CallContext) Memory(idx uint32) *Memory  // borrowed
func (c *CallContext) Module() *Module            // borrowed
func (c *CallContext) Executor() *Executor        // borrowed
```

Mechanics (the part that must not be improvised):

- One exported cgo trampoline (`//export wasmedgego_hostFuncInvoke`) is
  registered as the C function pointer for *every* host function; the
  per-function `Data` pointer carries a `cgo.Handle` to the Go closure.
- The trampoline `recover()`s panics and converts them into a
  `ErrCategoryUserLevel` failure — a Go panic never unwinds into C.
- Params/returns use `unsafe.Slice` views over the C arrays (no
  `reflect.SliceHeader`).
- The `Function` wrapper owns the handle; `Close`/module teardown releases
  it. Functions added to a `Module` are freed by the engine, but the Go-side
  handle is released by the owning `Module` wrapper on its `Close`.

### 5.6 Modules, Instances (`module.go`, `memory.go`, `table.go`, `global.go`, `tag.go`)

- `ASTModule` = loaded/compiled-but-not-instantiated module (from `Loader`).
  Exposes `Imports()`/`Exports()` metadata.
- `Module` = `WasmEdge_ModuleInstanceContext`. Both *host modules*
  (`NewModule("env")`) and *instantiated wasm* use this type, mirroring the
  C API.
- Host data: `NewModuleWithData(name, data any)` uses
  `ModuleInstanceCreateWithData` + a finalizer trampoline so the Go value is
  released when the engine deletes the module.
- `Memory` adopts 0.17's 64-bit lengths. Reads/writes are copy-based
  (`ReadAt`-style, bounds-checked by the engine) plus an explicitly
  documented zero-copy `UnsafeSlice(offset, len)` escape hatch whose
  validity ends at the next memory growth.
- `Limits` is a plain Go struct `{Min, Max uint64; HasMax, Shared, Is64 bool}`
  materialized to the C `LimitContext` only inside type constructors
  (same trick as `Config`).
- `Tag` types/instances (exception-handling proposal) are bound read-only,
  as the C API defines them.

### 5.7 Execution Pipeline (`loader.go`, `validator.go`, `executor.go`, `store.go`, `statistics.go`)

Direct, idiomatic wrappers:

```go
loader, _ := NewLoader(nil)
ast, err  := loader.LoadFile("app.wasm")     // or LoadBytes
val, _    := NewValidator(nil)
err        = val.Validate(ast)
exec, _   := NewExecutor(nil)                 // options: WithStats(stats)
store     := NewStore()
mod, err  := exec.Instantiate(store, ast)     // or Register(store, ast, name) / RegisterImport(store, m)
out, err  := exec.Invoke(fn, I32(1), I64(2))
```

- `Executor.InvokeContext(ctx, fn, params...)` uses `ExecutorAsyncInvoke` +
  `AsyncCancel` for cancellation/timeouts.
- `Statistics` is an owned object passed to `NewExecutor(WithStats(...))`;
  per-instruction cost tables and cost limits are bound 1:1.
- 0.17 additions `ExecutorRegisterImportWithAlias` and
  `LoaderSerializeASTModule` are included.

### 5.8 VM (`vm.go`, `async.go`)

```go
vm, err := NewVM(&Config{WASI: true})
defer vm.Close()

// One-shot
out, err := vm.RunFile("app.wasm", "fib", I32(30))
out, err  = vm.RunBytes(b, "fib", I32(30))

// Staged
err = vm.LoadFile("app.wasm"); err = vm.Validate(); err = vm.Instantiate()
out, err = vm.Execute("fib", I32(30))
out, err = vm.ExecuteRegistered("mod", "fn", ...)

// Cancellation — implemented over the C Async API
out, err = vm.ExecuteContext(ctx, "fib", I32(35))

// Advanced: promise form
ex := vm.ExecuteAsync("fib", I32(35))   // *Execution
ex.Cancel(); out, err := ex.Wait()      // Wait is Close-like: releases the C object
```

- Registration: `vm.RegisterModule(name, ...)` variants incl. 0.17's
  `...FromImportWithAlias`.
- Introspection: `vm.FunctionList()`, `vm.ActiveModule()`,
  `vm.RegisteredModule(name)` (borrowed), component accessors
  (`vm.Store()`, `vm.Loader()`, … all borrowed).
- `Execution` wraps `WasmEdge_Async` (owned; `Wait`/`WaitFor`/`Cancel`).

### 5.9 WASI (`wasi.go`)

```go
w, err := NewWASIModule(WASIConfig{
    Args: os.Args[1:], Envs: []string{"K=V"}, Preopens: []string{".:/"},
    // Optional fd redirection → ModuleInstanceCreateWASIWithFds
    Stdin, Stdout, Stderr *os.File
})
w.ExitCode()                 // uint32, after execution
w.NativeHandler(fd int32)    // (uint64, error)
```

When `Config.WASI` is set on a `VM`, `vm.WASIModule()` returns the built-in
instance (borrowed) for `InitWASI`-style reconfiguration.

### 5.10 Plugins (`plugin.go`), Logging (`log.go`), Compiler (`compiler.go`), Tools (`tools.go`)

- `LoadPluginsFromDefaultPaths()`, `LoadPlugins(path)`, `PluginNames()`,
  `FindPlugin(name) *Plugin`, `(*Plugin).ModuleNames()`,
  `(*Plugin).CreateModule(name) (*Module, error)`,
  `InitWASINN(preloads ...string)`.
- Logging: `SetLogLevel(LogLevelError)`, `SetLogOff()`,
  `SetLogCallback(func(LogMessage))` bridging `WasmEdge_LogSetCallback`,
  plus `RouteLogsToSlog(*slog.Logger)` — engine logs become structured Go
  logs (single global callback trampoline, same panic-containment rules as
  host functions).
- `Compiler`: `NewCompiler(cfg)`, `CompileFile(in, out)`,
  `CompileBytes(b, out)`.
- Driver entry points: `DriverCompiler(args)`, `DriverTool(args)`,
  `DriverUniTool(args)` (int exit codes, for building custom `wasmedge`-like
  CLIs).

### 5.11 Version Gating (`wasmedge.go`)

`Version() string`, `VersionMajor/Minor/Patch() uint32`. `init()` verifies
the loaded shared library's major.minor matches the headers the binding was
compiled against and panics with an actionable message on mismatch (fail
fast at process start, not at the first weird crash).

## 6. Naming Conventions (breaking)

| v1 | v2 |
|---|---|
| `NewConfigure(...)` + `Release` | `&Config{...}` (no lifetime) |
| `Release()` | `Close() error` (`io.Closer`, idempotent) |
| `Result`, `GetMessage()` | `error`, `*wasmedge.Error` |
| `interface{}` params/returns | `Value` (`I32(…)`, `v.I32()`) |
| `GetVersion()` | `Version()` (no `Get` prefixes anywhere) |
| `self *T` receivers | idiomatic short receivers |
| `VM.RunWasmFile(...)` | `VM.RunFile(...)` |
| `Executor.Invoke(store,...)` legacy shapes | typed, store-free `Invoke(fn, ...)` |
| build tags `image`, `tensorflow` | removed (plugins replace them) |

## 7. Testing & CI

- **Fixtures:** `internal/testwasm` builds tiny WASM binaries *in Go source*
  (hand-encoded sections with comments; no wat2wasm/network dependency):
  `AddModule` (i32 add), `FibModule`, `HostCallModule` (imports a host fn),
  `MemoryModule` (exports memory).
- **Tests:** table-driven unit tests per concept file; lifecycle tests
  (double-Close, use-after-Close, transfer-then-Close), host-function
  round-trips incl. panic containment, context cancellation with an infinite
  loop module, WASI exit-code path.
- **CI:** GitHub Actions matrix (ubuntu-24.04, macos-14) installing the
  matching WasmEdge release via the official install script; `go vet`,
  `golangci-lint`, `go test -race`. Local dev uses
  `WASMEDGE_DIR=~/workspace/WasmEdge/build` overrides (documented in
  CONTRIBUTING).

## 8. Rollout

1. This branch replaces the tree with the v2 module (v1 remains importable
   forever via the module proxy — that is the point of `/v2`).
2. Pre-releases tagged `v2.0.0-alpha.N` while intern TODOs are burned down.
3. `docs/MIGRATION.md` maps every v1 symbol to its v2 replacement (table
   seeded in this spec §6).
4. Bazel files from #58 are updated for the new layout in a follow-up
   (tracked in PLAN.md Phase 9, not silently dropped).

## 9. Open Questions (deliberately deferred)

- Whether to publish a `wasi_nn`-style typed plugin sub-package once the
  plugin C APIs stabilize (needs its own cgo package; feasible, but wait for
  demand).
- Threads proposal: shared memories are bindable today, but Go-side shared
  `[]byte` aliasing rules need a doc pass before we advertise it.
- `wasmedge_experimental.h` pre/post host hooks behind a build tag.

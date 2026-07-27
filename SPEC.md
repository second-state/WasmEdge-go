# WasmEdge-go v2 Redesign Specification

Status: ALPHA CANDIDATE (refreshed hosted matrix validation pending)
Author: WasmEdge-go maintainers
Updated: 2026-07-27
Target: WasmEdge C API >= 0.17.1 and < 0.18.0

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
2. Accounted coverage of the **stable** 0.17 C API surface
   (`wasmedge_basic/value/configure/ast/instance/execution/vm/compiler/plugin/tools`);
   see [docs/API_COVERAGE.md](docs/API_COVERAGE.md) for all 291 export
   declarations and every semantic/omission decision.
3. Memory safety by construction: idempotent `Close`, ownership tracking for
   caller-owned vs. borrowed vs. transferred objects, no way to double-free
   from safe Go code, panic containment at every cgo callback boundary.
4. A scaffolding-complete codebase where every remaining gap is documented
   with its upstream or platform constraint and acceptance test.

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
├── go.mod                                  go 1.25
├── *.go                                    package wasmedge (root package)
├── internal/testwasm/                      hand-encoded WASM fixtures (pure Go)
├── example_test.go                         runnable pkg.go.dev examples
├── testdata/consumer/                      out-of-tree API compatibility build
├── docs/                                   design principles and migration guide
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
- **Active Go releases only.** v2 currently supports Go 1.25 and 1.26; the
  `go` directive and CI matrix advance when Go's upstream support window does.

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
  `*wasmedge.Error` propagates its code; any other Go error crosses the engine
  as a private user-result token and is restored at the initiating Go call.
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
- `ExternRef` pins a Go value with `runtime/cgo.Handle`, but only a
  package-allocated opaque C token crosses the C boundary. It is an owned
  resource: `NewExternRef(v any) *ExternRef`, `Value() any`, `Close() error`.
  Reference `Value`s remember their owner for validation and rooting; they do
  not make an explicit early owner `Close` safe.
- V128 is `[16]byte` (little-endian lane order), not `big.Int`.

### 5.3 Ownership and `Close` (`resource.go`)

`Config`, `Limits`, `FunctionType`, `TableType`, `MemoryType`, `GlobalType`,
and `TagType` are plain Go descriptors. They are copied when crossing the C
boundary and never need `Close`.

Every native wrapper embeds a small `lifetime` helper that records:

- **owned** — created by us, `Close` calls the C destructor. Idempotent
  (second `Close` is a no-op returning nil). All owned wrappers implement
  `io.Closer`.
- **borrowed** — obtained from a container (e.g. `Store.Module`,
  `Module.Function`, `CallContext.Memory`). `Close` is a no-op; validity is
  bounded by the owner and documented on the accessor. Borrowed views hold
  a strong reference to their owning wrapper and recursively check its
  validity (views scoped to a host call are instead guarded by the
  `CallContext` validity flag).
- **transferred** — ownership moved into the engine (e.g. a `*Function`
  added to a `*Module` via `AddFunction`, or a standalone table/memory/global
  added via `Module.Add*`). The transfer API disarms the wrapper so a later
  `Close` cannot double-free.

Dependencies held through native non-owning pointers use leases. For example,
a VM leases an external `Store`, an Executor leases its `Statistics`, and a
VM leases registered import modules. An instantiated module separately leases
the Store modules it linked against. Closing a leased dependency returns
`ErrInUse`. VM registered-import leases end at `VM.Reset`/`VM.Close`;
external Store and Statistics leases end only when their VM/Executor closes.
A direct Store registration is a strong provenance edge, not a lease:
`Module.Close` uses WasmEdge's automatic unregistration and clears the
matching Go registry entries unless an instantiated dependant still leases
the module. Registration APIs require a caller-owned `Module`; borrowed views
are rejected with `ErrOwnership` because their original owner could otherwise
invalidate an alias that the Store still exposes.

Reference arguments use a prepare/commit/rollback transaction. The binding
acquires every new lease before native entry, rolls the whole batch back when
setup fails, and commits it only after a synchronous call entered native code
or an async object was created. Classic externref/funcref mismatches are
rejected before preparation; indexed-reference subtyping remains an engine
check because its graph is private to the defining module.

WasmEdge 0.17.1's `VMCleanup` also has an important external-Store effect:
`VM.Reset` clears all non-built-in registrations from its Store. With an
external Store this includes registrations that predate the VM, and the Go
binding clears their Go retention entries to mirror the native registry.
`VM.Close`, in contrast, destroys VM-owned guest registrations but preserves
both pre-existing registrations and caller-owned imports/aliases in an
external Store. The latter remain direct Store registrations until their
module is closed.

VM-owned module views are also generation-scoped. A successful staged
`Instantiate`, a started one-shot `Run*`, or `Reset` invalidates affected
active or registered views even though the VM wrapper itself remains open.
Using an invalidated borrowed view is a programmer error and panics.

Native teardown is explicit-only. Owned wrappers register a
`runtime.AddCleanup` solely to report a missing `Close` under
`-tags wasmedge_debug`; the cleanup deliberately does not invoke C because Go
does not order cleanups across dependent native objects. In normal builds it
is a no-op. An omitted `Close` therefore leaks the native resource rather
than risking nondeterministic use-after-free.

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
    WASI             bool       // built-in host registration; initially no fd mappings
    RunMode          RunMode    // 0 = Interpreter (engine default)
    MaxMemoryPages   uint64     // 0 ⇒ engine default
    AllowAFUNIX      bool
    Compiler         CompilerConfig // OptLevel, OutputFormat, DumpIR, GenericBinary, Interruptible
    Stats            StatsConfig    // InstructionCounting, CostMeasuring, TimeMeasuring
}
```

Constructors accept `*Config` where nil means defaults:
`NewVM(nil)`, `NewVM(&Config{RunMode: RunModeJIT})`.
Internally `(*Config).build()` materializes a `WasmEdge_ConfigureContext`,
the constructor consumes it, and it is deleted before returning. Unknown
enum values are rejected with an error matching `ErrInvalidArgument`; native
configuration allocation failure matches `ErrUnavailable` rather than
silently falling back to defaults.

New 0.17 knobs (`Standard` WASM_1/2/3, `RunMode` Interpreter/JIT/AOT,
`AllowAFUNIX`) are first-class fields.

`Config.Effective() (EffectiveConfig, error)` materializes the declaration,
queries every native configuration getter, and returns the complete enabled
proposal set plus the actual defaults/overrides as ordinary Go data. A nil
receiver is valid and reports WasmEdge defaults. For 0.17.1, the compiler
defaults are O3 and universal Wasm output, not native output.

### 5.5 Host Functions (`function.go`, `callcontext.go`)

Two levels:

```go
// Level 1: explicit — full control, mirrors the C API.
type HostFunc func(call *CallContext, params []Value) ([]Value, error)
func NewFunction(ft FunctionType, fn HostFunc, opts ...FunctionOption) (*Function, error)

// Level 2: reflective — the common path. Derives the FunctionType from the
// Go signature. Supported: int32/uint32/int64/uint64/float32/float64,
// [16]byte, *ExternRef, *Function, optional leading *CallContext, and an
// optional trailing error.
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
  registered as the C function pointer for *every* host function. Each
  binding carries a package-allocated opaque C token in `This`; `Data` is
  null. The trampoline resolves that token through a package-private
  registry whose Go entry owns the callback. Neither a Go pointer nor a
  numeric `cgo.Handle` value crosses the C boundary.
- The trampoline `recover()`s panics and converts them into a
  `ErrCategoryUserLevel` failure — a Go panic never unwinds into C.
- Params/returns use `unsafe.Slice` views over the C arrays (no
  `reflect.SliceHeader`).
- The `Function` wrapper owns the handle; `Close`/module teardown releases
  it. Functions added to a `Module` are freed by the engine, but the Go-side
  handle is released by the owning `Module` wrapper on its `Close`.
- Host funcref result roots use the canonical native target, not only the
  wrapper that carried it. A result that would reverse an existing
  Module/VM import dependency or close a transitive host-function root cycle
  returns `ErrReferenceCycle`.
- `CallContext` and every borrowed view derived from it expire when the host
  callback returns. They must not escape to another goroutine or be retained
  for later use.

### 5.6 Modules, Instances (`module.go`, `memory.go`, `table.go`, `global.go`, `tag.go`)

- `ASTModule` = loaded/compiled-but-not-instantiated module (from `Loader`).
  Exposes `Imports()`/`Exports()` metadata.
- `Module` = `WasmEdge_ModuleInstanceContext`. Both *host modules*
  (`NewModule("env")`) and *instantiated wasm* use this type, mirroring the
  C API.
- Host data: `NewModuleWithData(name, data any)` uses
  `ModuleInstanceCreateWithData` plus a native finalizer callback and an
  opaque C token so the Go value is released when the engine deletes the
  module. This is native ownership finalization, not GC-based wrapper
  cleanup.
- `Memory` adopts 0.17's 64-bit lengths. Reads/writes are copy-based
  (`Read`/`Write`, range-checked by the binding before native entry) plus an explicitly
  documented zero-copy `UnsafeSlice(offset, len)` escape hatch whose
  validity ends at the next memory growth or instance close. Shared memory
  rejects `UnsafeSlice` with `ErrUnsafeSharedMemory`. Construction rejects a
  memory64 minimum whose byte size overflows the native allocator and verifies
  the allocated page count before returning success.
- `Limits` is a plain Go struct `{Min, Max uint64; HasMax, Shared, Is64 bool}`
  materialized to the C `LimitContext` only while a resource constructor
  consumes its enclosing descriptor (same trick as `Config`). Validation
  enforces the proposal bounds: memory32 at most 2^16 pages, memory64 at most
  2^48 pages, and table32 at most 2^32-1 elements. Standalone table
  construction/growth also rejects a concrete size beyond the host index
  range before entering WasmEdge 0.17.1's `noexcept` vector allocation path.
- `FunctionType`, `TableType`, `MemoryType`, `GlobalType`, and `TagType` are
  plain descriptors. `NewFunction`, `NewTable`, `NewTableWithInit`,
  `NewMemory`, and `NewGlobal` return `(*T, error)` and classify invalid
  descriptors/initializers as `ErrInvalidArgument`.
- Reference-valued table/global initializers and setters must exactly match
  the declared `ValType`; null is rejected for non-nullable reference types
  before any native mutation.
- `NewModule` is allocation-only and semantically infallible; it returns one
  result and panics only if WasmEdge cannot allocate its native context.
- `Tag` types/instances (exception-handling proposal) are bound read-only,
  as the C API defines them.

### 5.7 Execution Pipeline (`loader.go`, `validator.go`, `executor.go`, `store.go`, `statistics.go`)

Direct, idiomatic wrappers:

```go
loader, err := NewLoader(nil)
if err != nil { /* handle */ }
defer loader.Close()
ast, err := loader.LoadFile("app.wasm") // or LoadBytes
if err != nil { /* handle */ }
defer ast.Close()
validator, err := NewValidator(nil)
if err != nil { /* handle */ }
defer validator.Close()
err = validator.Validate(ast)
executor, err := NewExecutor(nil) // options: WithStats(stats)
if err != nil { /* handle */ }
defer executor.Close()
store := NewStore()
defer store.Close()
module, err := executor.Instantiate(store, ast)
if err != nil { /* handle */ }
defer module.Close()
fn, ok := module.Function("run")
if !ok { /* handle missing export */ }
out, err := executor.Invoke(fn, I32(1), I64(2))
```

- `Executor.InvokeAsync(fn, params...)` exposes the manually managed
  `WasmEdge_Async` path; `InvokeContext(ctx, fn, params...)` builds on it with
  `AsyncCancel` for cancellation/timeouts.
- `Statistics` is an owned object passed to
  `NewExecutor(nil, WithStats(stats))`;
  per-instruction cost tables and cost limits are bound 1:1. The Executor
  leases it until `Executor.Close`, so an early `Statistics.Close` reports
  `ErrInUse`.
- `NewStore` and `NewStatistics` are allocation-only, one-result
  constructors; they panic only on impossible native allocation failure.
- 0.17 additions `ExecutorRegisterImportWithAlias` and
  `LoaderSerializeASTModule` are included.

The official 0.17.1 `darwin/arm64` artifact can abort the process in
`LoaderSerializeASTModule`. The binding therefore returns
`ErrSerializeUnsupported` (wrapping `errors.ErrUnsupported`) before native
entry on that platform. Other platforms keep the native serialization gate
enabled.

### 5.8 VM (`vm.go`, `async.go`)

```go
vm, err := NewVM(nil)
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
ex, err := vm.ExecuteAsync("fib", I32(35))
if err != nil { /* handle */ }
_ = ex.Cancel()
out, err = ex.Wait() // terminal: collects results and releases the C object

// Reset is fallible and invalidates active/registered borrowed views.
err = vm.Reset()
```

- Registration: `RegisterModule`, `RegisterModuleBytes`,
  `RegisterModuleFile`, `RegisterImport`, and 0.17's
  `RegisterImportWithAlias`.
- Introspection: `vm.Functions()`, `vm.ActiveModule()`,
  `vm.RegisteredModule(name)` (borrowed), component accessors
  (`vm.Store()`, `vm.Loader()`, … all borrowed).
- `Execution` wraps `WasmEdge_Async`. Exactly one of `Wait` or `Close` is the
  terminal consumer. `WaitFor` only observes completion; `Cancel` only
  requests interruption and may race the waiting path.
- VM and Executor each permit one execution at a time; overlap reports
  `ErrInUse`. Context-aware methods cancel and drain before returning.
- WasmEdge 0.17.1 shares a stop token across executions. If cancellation
  races native completion without consuming that token, the terminal
  operation reports `ErrCancellationRace` and poisons the originating
  wrapper; later executions report `ErrUnusable` and the VM/Executor must be
  recreated. Context methods return the context error after draining instead
  of exposing `ErrCancellationRace`, but still poison the wrapper.
- Active-module borrowed views are generation-bound. Successful
  `Instantiate`, every started one-shot `Run*`, and `Reset` invalidate the
  prior generation. `Reset() error` also invalidates registered views and
  returns `ErrInUse` rather than tearing down resources leased by an
  in-flight async operation.

### 5.9 WASI (`wasi.go`)

```go
w, err := NewWASIModule(WASIConfig{
    Args: os.Args[1:], Envs: []string{"K=V"}, Preopens: []string{".:/"},
    // Explicit Unix fd capability → ModuleInstanceCreateWASIWithFds
    Stdio: RedirectWASIStdio(in, out, errOut),
})
if err != nil { /* handle configuration/allocation failure */ }
defer w.Close()
code, err := w.WASIExitCode()    // (uint32, error), after execution
handler, err := w.WASINativeHandler(fd) // (uint64, error)
```

When `Config.WASI` is set on a `VM`, `vm.WASIModule()` returns the built-in
instance (borrowed) for initialization/reconfiguration. Registration alone
leaves the native fd table empty. The first `InitWASI` must explicitly use
`DiscardWASIStdio()`, `InheritWASIStdio()`, or
`RedirectWASIStdio(...)`; a zero `WASIStdio` returns
`ErrWASIStdioPolicyRequired`, matching `ErrInvalidArgument`, before native
state changes. Discard supplies EOF on stdin and output sinks using
binding-owned null-device descriptors, so args/env/preopens do not require
ambient stdio access. The instance belongs to the VM's registered generation;
reacquire it after `Reset`.
WASI-only methods accept only modules carrying private provenance from
`NewWASIModule` or `VM.WASIModule`; a normal module cannot opt in by using
the `wasi_snapshot_preview1` name. `InheritWASIStdio` is the deliberate host
capability that grants the embedding process's streams; there is no implicit
nil-to-inherit behavior. Redirected files remain caller-owned, but the module
retains their `*os.File` wrappers until module/generation teardown so garbage
collection cannot close a descriptor still used by WasmEdge.
WasmEdge 0.17.1 cannot replace initialized stdio/preopen fd-table entries:
later `InitWASI` calls may change args/env and reset exit state, but must
repeat the exact original `Preopens`, policy, and files. Mapping changes
return `ErrWASIResourceMappingImmutable` and wrap `errors.ErrUnsupported`
before native state changes. On Unix, redirected descriptors must also still
be open and fit `int32`; a failed reconfiguration preserves the previous
descriptor roots. Redirection is rejected on Windows because the WasmEdge C
API expects CRT file descriptors while `os.File.Fd` exposes a Windows handle.
`DiscardWASIStdio` is supported on Windows through binding-owned `NUL`
handles converted by the same UCRT stdio API-set used by the official
WasmEdge DLL; initialization verifies the exact native HANDLE mappings and
teardown closes the descriptors through that same UCRT.

### 5.10 Plugins (`plugin.go`), Logging (`log.go`), Compiler (`compiler.go`), Tools (`tools.go`)

- `LoadPluginsFromDefaultPaths()`, `LoadPlugins(path) error`, `PluginNames()`,
  `FindPlugin(name) (*Plugin, bool)`, `(*Plugin).ModuleNames()`,
  `(*Plugin).CreateModule(name) (*Module, error)`,
  `InitWASINN(preloads []string) error`,
  `InitWasmEdgeProcess(allowedCmds []string, allowAll bool) error`.
- Logging: `SetLogLevel(LogLevelError)`, `SetLogOff()`,
  `SetLogCallback(func(LogMessage))` bridging `WasmEdge_LogSetCallback`,
  plus `RouteLogsToSlog(*slog.Logger)` — engine logs become structured Go
  logs (single global callback trampoline, same panic-containment rules as
  host functions).
- `Compiler`: `NewCompiler(cfg)`, `CompileFile(in, out)`,
  `CompileBytes(b, out)`.
- Driver entry points: `DriverCompiler(args)`, `DriverTool(args)`,
  `DriverUniTool(args)` (int exit codes, for building custom `wasmedge`-like
  CLIs). They return exit code 2 without entering the native driver when an
  argument contains an embedded NUL byte. On Windows they first configure the
  console output code page for UTF-8.

`WasmEdge_Driver_WasiNNRPCServer` is not linked because official 0.17.1 SDKs
do not export that build-conditional symbol. A future binding requires a
dedicated build tag, feature-matched custom runtime, and CI lane. The unsafe
`WasmEdge_VMForceDeleteRegisteredModule` and provider-side
`WasmEdge_Plugin_GetDescriptor` decisions are also documented in
[docs/API_COVERAGE.md](docs/API_COVERAGE.md).

Every public value passed to a native NUL-terminated `char *` is validated
before allocation or VM state mutation. Embedded NUL bytes match
`ErrInvalidArgument`; this covers file/plugin paths, WASI arguments,
environment entries and preopens, plugin/process lists, and driver argv.
Length-delimited `WasmEdge_String` names preserve their full contents and do
not use this restriction.

### 5.11 Version Gating (`wasmedge.go`)

`Version() string`, `VersionMajor/Minor/Patch() uint32`. `init()` verifies
the loaded shared library's major.minor matches the headers the binding was
compiled against and panics with an actionable message on mismatch (fail
fast at process start, not at the first weird crash).

## 6. Naming Conventions (breaking)

| v1 | v2 |
|---|---|
| `NewConfigure(...)` + `Release` | `&Config{...}` (no lifetime) |
| heap-backed `New*Type` + `Release` | plain `FunctionType`/`TableType`/`MemoryType`/`GlobalType`/`TagType` values |
| `Release()` | `Close() error` (`io.Closer`, idempotent) |
| `Result`, `GetMessage()` | `error`, `*wasmedge.Error` |
| `interface{}` params/returns | `Value` (`I32(…)`, `v.I32()`) |
| `GetVersion()` | `Version()` (no `Get` prefixes anywhere) |
| `self *T` receivers | idiomatic short receivers |
| `VM.RunWasmFile(...)` | `VM.RunFile(...)` |
| `VM.Cleanup()` | `VM.Reset() error` |
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
- **CI:** GitHub Actions matrix (ubuntu-24.04, macos-15) plus a configured
  `windows-2025`/Go 1.25–1.26 lanes, all installing checksum-pinned official
  WasmEdge 0.17.1 artifacts; `go build`, `go vet`, `golangci-lint`,
  `go test -race`, `GOEXPERIMENT=cgocheck2`, leak-debug tests, ABI relocation
  checks, an external consumer build, and Bazel system-SDK tests on each host
  OS. The refreshed matrix awaits an exact-commit hosted validation; Windows
  has not yet had a hosted run. Local dev uses
  `WASMEDGE_DIR=~/workspace/WasmEdge/build` overrides (documented in
  CONTRIBUTING).
- **Bazel:** Bazel 9.2.0/rules_go 0.62.0 use a caller-selected system SDK via
  `--repo_env=WASMEDGE_SDK=/absolute/prefix`; `bazel test ... //...` covers the
  package, examples, and external consumer target without downloading a
  native runtime. CI runs this path once per supported host OS on the Go 1.26
  lane.

## 8. Rollout

1. This branch replaces the tree with the v2 module (v1 remains importable
   forever via the module proxy — that is the point of `/v2`).
2. Pre-releases tagged `v2.0.0-alpha.N` while platform gates and
   stabilization findings are burned down.
3. `docs/MIGRATION.md` maps common v1 workflows to v2 and identifies
   intentionally removed API families; the declaration-level C API audit is
   maintained separately in `docs/API_COVERAGE.md`.
4. Bazel files are restored for the root-package layout. The repository rule
   consumes an explicit WasmEdge SDK prefix, so Bazel and Go builds can test
   the same native artifact.

## 9. Open Questions (deliberately deferred)

- Whether to publish a `wasi_nn`-style typed plugin sub-package once the
  plugin C APIs stabilize (needs its own cgo package; feasible, but wait for
  demand).
- Whether a future Go API can expose shared memory without violating the Go
  memory model. v2 currently supports copy-based `Read`/`Write` and rejects
  `UnsafeSlice` for shared memory.
- Whether an upstream WasmEdge release can eliminate the shared stop-token
  cancellation race so a later API line can remove the `ErrUnusable`
  recovery requirement.
- `wasmedge_experimental.h` pre/post host hooks behind a build tag.

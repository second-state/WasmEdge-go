# WasmEdge-go v2 Design Principles

Status: living design for the v2 alpha candidate

Runtime target: WasmEdge C API >= 0.17.1 and < 0.18.0

Go target: active supported Go releases (currently Go 1.25 and 1.26)

This document explains the user-facing design decisions behind v2. It is
shorter and more normative than [SPEC.md](../SPEC.md): the specification
describes the intended API surface, while this document records the rules
that should guide implementation and review.

## Runtime and ABI boundary

v2 deliberately targets one stable WasmEdge C API line. It is compiled
against 0.17.1 headers and rejects a loaded runtime outside the supported
`>= 0.17.1, < 0.18.0` range. This is an ABI safety boundary, not merely a
feature check.

The stable header surface is accounted for declaration by declaration in
[API_COVERAGE.md](API_COVERAGE.md): 291 export declarations have an explicit
direct-binding, semantic-equivalent, omission, or provider-side decision.

WasmEdge 0.17.1 added ELF symbol versions to five functions whose ABI changed
in 0.17:

- `WasmEdge_LimitIsEqual`
- `WasmEdge_TableTypeCreate`
- `WasmEdge_TableTypeGetLimit`
- `WasmEdge_MemoryTypeCreate`
- `WasmEdge_MemoryTypeGetLimit`

Consequently, Linux and Android binaries must be rebuilt after replacing a
0.17.0 runtime with 0.17.1. Reusing an object linked against 0.17.0 can select
the compatibility symbol rather than the new 0.17 ABI. Release CI must compile
against 0.17.1 and verify that relocations for these functions resolve to
`WASMEDGE_0.17`.

The relevant upstream sources are the
[WasmEdge 0.17.1 release](https://github.com/WasmEdge/WasmEdge/releases/tag/0.17.1),
the [ABI correction pull request](https://github.com/WasmEdge/WasmEdge/pull/4951),
and the
[0.17.1 compatibility implementation](https://github.com/WasmEdge/WasmEdge/blob/0.17.1/lib/api/wasmedge_compat.cpp).

## One package, two ways to use it

The package serves two audiences without making either learn the other's
workflow.

### The common path: `VM`

`VM` owns the load, validate, instantiate, and execute state machine. A Go
application that wants to run a module should need one primary object:

```go
vm, err := wasmedge.NewVM(&wasmedge.Config{WASI: true})
if err != nil {
	return err
}
defer vm.Close()

results, err := vm.RunBytes(wasm, "fib", wasmedge.I32(30))
```

Registration, staged execution, and cancellation remain available on the
same object, but are not prerequisites for the first successful program.

### The embedding path: explicit pipeline

Applications that cache ASTs, control stores, collect statistics, or manage
host modules can use the engine pipeline directly:

```go
loader, err := wasmedge.NewLoader(nil)
if err != nil {
	return err
}
defer loader.Close()

ast, err := loader.LoadBytes(wasm)
if err != nil {
	return err
}
defer ast.Close()

validator, err := wasmedge.NewValidator(nil)
if err != nil {
	return err
}
defer validator.Close()
if err := validator.Validate(ast); err != nil {
	return err
}

store := wasmedge.NewStore()
defer store.Close()
executor, err := wasmedge.NewExecutor(nil)
if err != nil {
	return err
}
defer executor.Close()

module, err := executor.Instantiate(store, ast)
if err != nil {
	return err
}
defer module.Close()
fn, ok := module.Function("fib")
if !ok {
	return errors.New("fib export not found")
}
results, err := executor.Invoke(fn, wasmedge.I32(30))
```

The explicit layer follows upstream concepts closely enough that WasmEdge C
documentation remains useful. The high-level layer removes ceremony; it does
not introduce a second execution engine or a conflicting object model.

## Go-facing contracts

### Typed values, not `any`

WASM values cross the public execution API as `Value`. Constructors such as
`I32`, `F64`, `V128`, `ExternRefValue`, and `FuncRefValue` make the type
explicit. Accessors such as `Value.I32` validate the kind. A kind mismatch is
a programmer error and panics, like an invalid `reflect.Value` accessor;
module loading, validation, and execution failures return `error`.

Reference values created by this binding carry their Go owner so containers
and in-flight executions can validate and lease it. A standalone `Value`
does not authorize an explicit early owner `Close`; callers keep the
`ExternRef` or host `Function` open while the value can be observed. This is
not an invitation to pass arbitrary Go pointers through C. Foreign reference
tokens are never interpreted as package-owned Go handles.
Table and global construction/mutation also validates the exact declared
`ValType` before native entry and rejects null for non-nullable reference
types.

### Errors compose with the standard library

Engine failures are `*wasmedge.Error` values carrying an error category, code,
and captured message. Callers inspect them with `errors.Is` and `errors.As`.
Binding-level failures use stable sentinels such as `ErrClosed`,
`ErrOwnership`, and `ErrAlreadyExists`, wrapped with operation context.

Graceful WasmEdge termination is not a failure in the C API, so the
`Terminate` host-function sentinel stops execution without producing a
non-nil execution error. WASI exit status is read from the WASI module.

Constructors that validate caller input return useful errors:
`NewFunction`, `NewTable`, `NewTableWithInit`, `NewMemory`, and `NewGlobal`
return `(*T, error)` and match `ErrInvalidArgument` for invalid descriptors or
initializers. Pipeline, VM, Compiler, and WASI constructors also return
errors. Allocation-only `NewModule`, `NewStore`, and `NewStatistics` are
documented as semantically infallible and panic only if WasmEdge cannot
allocate their native context.

`Config` remains declarative, while `Config.Effective()` makes the result
inspectable. It asks the native 0.17.1 getters for the complete enabled
proposal set and all effective defaults/overrides, then returns ordinary Go
data with no native lifetime. A nil receiver reports engine defaults; the
0.17.1 compiler defaults are O3 and universal Wasm output.

Host callbacks must never unwind a Go panic through cgo. The trampoline
recovers panics as `HostPanicError`, including the Go stack. Ordinary Go host
errors receive a private user-result token while crossing WasmEdge and are
restored at the initiating Go invocation, preserving `errors.Is`,
`errors.As`, wrapping, and the original message. A token has no wall-clock
expiry: `Execution.Wait` consumes it even after an arbitrarily long delay,
while `Execution.Close` drains and discards the result before deleting the
native async object.

### Cancellation follows `context.Context`

Synchronous convenience methods have context-aware counterparts. They start a
WasmEdge async execution, request engine interruption when the context is
done, and wait for the native operation to settle. When cancellation wins,
they return `context.Canceled` or `context.DeadlineExceeded`; native
completion may win a simultaneous race and return its result.

`Execution`, created by `VM.ExecuteAsync` or `Executor.InvokeAsync`, exposes
the lower-level promise form for callers that need `WaitFor`, `Cancel`, or
explicit abandonment. Cancellation is an interruption request; a long-running
Go host function must still cooperate before it can return promptly.
`CallContext.Context()` exposes the exact invocation context to host code for
`InvokeContext`, `ExecuteContext`, and `RunContext` calls.
Synchronous and manually managed asynchronous calls expose
`context.Background()`. The binding is installed before native execution can
enter a callback and is removed only after the worker has fully settled.

Exactly one terminal consumer calls `Execution.Wait` or `Execution.Close`.
`WaitFor` is observational and `Cancel` does not release the native async
object. `Close` always drains a completed result, including a private host
error that the caller intentionally abandons. VM and Executor serialize
execution; overlap returns `ErrInUse`.

WasmEdge 0.17.1 uses a shared native stop token. If cancellation races a
completed invocation without the engine consuming the token, manual
`Wait`/`Close` returns `ErrCancellationRace` and marks the originating VM or
Executor unusable. Subsequent execution returns `ErrUnusable`; recreate the
wrapper. Context methods return the context's cancellation reason after
draining, but the same recreation requirement applies.

### Memory is safe by default

`Memory.Read` and `Memory.Write` copy data and enforce native memory bounds.
`Read` rejects a length that cannot be represented by a Go slice before
allocating. `NewMemory` also rejects a memory64 minimum whose byte size cannot
fit the 0.17.1 allocator and verifies that the native page count equals the
requested minimum before reporting success. They are the default API for
application code.

`Memory.UnsafeSlice` is an intentionally loud escape hatch. Its result aliases
engine-owned storage, becomes invalid after memory growth or instance close,
and makes writes immediately visible to WASM. It must not be retained across
calls that can grow memory. Shared memory is rejected with
`ErrUnsafeSharedMemory`: native threads can mutate it outside Go's memory
model and race detector, so callers use the copy-based `Read` and `Write`
methods instead.

## Ownership and concurrency

Native teardown is explicit-only. `runtime.AddCleanup` reports an accidentally
forgotten owned wrapper under `-tags wasmedge_debug`, but deliberately never
calls a C destructor: Go does not order cleanups across dependent native
objects, and guessing an order could turn a leak into a use-after-free.
Normal-build cleanup is a no-op. A forgotten `Close` therefore leaks.

| Kind | Examples | Owner and `Close` behavior | Concurrency contract |
|---|---|---|---|
| Plain descriptors and scalar values | `Config`, `Limits`, `FunctionType`, `TableType`, `MemoryType`, `GlobalType`, `TagType`, numeric `Value`, `ValType` | No C lifetime. Copy freely; descriptor slices are copied when crossing the native boundary. Do not mutate a shared `Config` while a constructor reads it. | Independent copies are safe. Ordinary Go data-race rules apply to mutable structs and slices. |
| Owned wrappers | `VM`, `Loader`, `Executor`, `Store`, `ASTModule`, standalone instances | The wrapper destroys its C context only on explicit `Close`. `Close` is idempotent. Shared internal lifetime state prevents a value copy from deleting or transferring the same C context twice. Use after close is a contract violation. | Pass wrapper pointers; do not copy wrapper struct values. The shared lifetime state is a defensive ownership guard, not shared synchronization for every wrapper field. Unless a method explicitly says otherwise, serialize operations on one wrapper and never race `Close` with use. |
| Borrowed views | `Module.Function`, `Store.Module`, VM component accessors | `Close` is a no-op. The view is bounded by its owner. A `CallContext` view expires when the callback returns. VM-owned module views also expire when their active/registered generation changes; an external caller-owned module follows that module's lifetime. | It inherits the owner's synchronization. A `CallContext` and its views must not escape to another goroutine or beyond the callback. |
| Transferred wrappers | a `Function`, `Table`, `Memory`, or `Global` passed to `Module.Add*` | On success the module owns the C object. The original wrapper is disarmed and must not be used again. Borrowed objects cannot be transferred. | Build or mutate a host module from one goroutine. Registration and `Add*` are not concurrent mutation APIs. |
| Retained dependencies | external `Store` on a `VM`, `Statistics` on an `Executor`, VM-registered imports, instance import dependencies | The dependant holds a strong Go reference and a lease while C retains a non-owning pointer. An early dependency `Close` returns `ErrInUse`. | Close/reset dependants before dependencies; synchronize through the parent contract. |
| In-flight execution | `Execution` | Exactly one terminal consumer calls `Wait` or `Close`; both settle and release the native async object. `WaitFor` is observational and `Cancel` only requests interruption. | `Cancel` may race the waiting path as used by context cancellation. Do not run multiple `Wait`/`Close` consumers. |
| Callbacks | `HostFunc`, log callback | The function/module or process-level registration retains the callback. Panics are contained at the cgo boundary. | The engine may invoke callbacks from arbitrary threads. Callback state is the application's synchronization responsibility. |

Lease release follows the native edge: `VM.Reset` releases registered-import
leases, while an external Store remains leased until `VM.Close`; Statistics
remains leased until `Executor.Close`; instantiated modules lease the imports
they linked against until they close. A direct Store registration only keeps
the Go wrapper reachable: WasmEdge automatically unregisters it when
`Module.Close` succeeds. Registration requires a caller-owned module;
borrowed views are rejected because their upstream owner could invalidate a
still-registered alias. For an external Store attached to a VM, WasmEdge
0.17.1 `VMCleanup` removes every non-built-in registration, even one that
predates the VM, so `VM.Reset` clears those Go retention entries too.
`VM.Close` destroys VM-owned guest registrations but preserves pre-existing
registrations and caller-owned imports/aliases in an external Store.

Invocation references are rooted transactionally: preparation acquires all
new leases without publishing them, setup failure rolls back the whole batch,
and a native start commits them atomically. Host-function result roots record
canonical native funcref targets and the Module/VM import graph. A reverse or
transitive lease edge that would make both sides permanently `ErrInUse`
returns `ErrReferenceCycle` instead.

Shared closed/lease-state tracking guarantees that an explicitly closed owned
C object is freed once, even if application code accidentally copied the
wrapper value. It does not make value-copying a supported way to fork wrapper
state, nor make a method racing with `Close` safe.
Different, independent engine objects may be used from different goroutines
when the upstream WasmEdge operation permits it and the application does not
share unsynchronized Go state.

VM module views carry a generation guard in addition to the VM lifetime.
A successful staged `Instantiate` advances the active generation. A started
one-shot `Run*` advances it before native execution because WasmEdge does not
report whether a failed workflow already replaced the module; old reference
roots are retained conservatively until reset or close. `VM.Reset() error`
invalidates both active and registered generations, releases registration
leases, and returns `ErrInUse` instead of resetting while an asynchronous
operation or borrowed dependency is leased.

## WASI is an explicit capability

WASI remains disabled unless the caller enables the VM registration or
constructs a WASI module. Arguments, environment entries, and preopened paths
are explicit lists; the binding must not silently inherit the process
environment, command line, or current directory. An empty `Preopens` list
means no filesystem preopens.

These strings, file/plugin paths, process and plugin lists, and driver argv
cross native APIs as NUL-terminated C strings. The binding rejects embedded
NUL bytes with `ErrInvalidArgument` before native entry or VM generation
mutation, rather than silently truncating a path or capability declaration.
Driver entry points preserve their integer-return API and use exit code 2 for
this invalid argv case.

This is the default-deny direction for the stable API:

- filesystem access comes only from explicit guest-to-host preopens;
- environment variables and arguments are passed explicitly;
- socket and process/plugin capabilities stay disabled unless configured;
- `wasmedge_process` uses an explicit command allowlist unless the caller
  consciously selects its `allowAll` mode;
- enabling the VM's WASI registration leaves its fd table empty;
- every WASI initialization requires an explicit `WASIStdio` value:
  `DiscardWASIStdio` supplies non-ambient EOF/output sinks,
  `InheritWASIStdio` grants the process streams, and `RedirectWASIStdio`
  grants exactly three caller-owned files. The zero value is rejected.

WASI-only operations use private constructor/accessor provenance, never the
forgeable module name, before passing a module pointer to a WASI-specific C
API. Redirected files remain caller-owned, while the module retains their Go
wrappers for the whole native-use interval. Discard descriptors are
binding-owned. Both are released only after native module/generation teardown
on `Reset` or `Close`. WasmEdge 0.17.1 cannot replace existing stdio or
preopen entries during re-init. The first `InitWASI` on a VM module
establishes those mappings; later calls may update args/env and reset exit
state only when `Preopens`, the stdio policy, and the exact redirected files
are unchanged. A change returns `ErrWASIResourceMappingImmutable`, which
wraps `errors.ErrUnsupported`, before native state changes. On Unix the
binding first verifies that every redirected descriptor is open and
representable as an `int32`. Windows redirection is rejected because
`os.File.Fd` exposes a HANDLE rather than WasmEdge's expected CRT descriptor.
The binding-owned discard policy remains available there: it opens `NUL`
handles, converts them through the same UCRT stdio API-set used by the
official WasmEdge DLL, and verifies the exact native HANDLE mappings after
initialization. It never passes a MinGW CRT descriptor into WasmEdge's
separate UCRT fd table.

The binding currently targets WasmEdge's stable WASI module surface. It does
not claim to provide a general capability-security proof; embedders must still
choose and audit the capabilities they expose.

## Known upstream/platform constraints

The official WasmEdge 0.17.1 `darwin/arm64` artifact can abort the process in
`Loader.Serialize`; this cannot be recovered after entering C. The binding
therefore rejects that call before native entry with
`ErrSerializeUnsupported` (wrapping `errors.ErrUnsupported`). Other platforms
keep the native call and round-trip gate enabled.

WasmEdge 0.17.1's table implementation performs vector construction and
growth in `noexcept` code. The binding rejects table sizes that exceed the
host's index range before native entry, avoiding deterministic overflow and
length-error termination. As with very large Go allocations, ordinary
resource exhaustion remains an upstream/process limit rather than a
recoverable guarantee.

The WASI-NN RPC driver is conditional on an upstream build macro and is absent
from official 0.17.1 SDKs. The exact omission and the requirements for a future
build-tagged binding are recorded in
[API_COVERAGE.md](API_COVERAGE.md#wasi-nn-rpc-driver).

## Component Model boundary

WasmEdge 0.17.1 exposes Component-related proposal and type-code constants,
but not a stable C runtime surface on which this binding can build component
instantiation and execution. v2 may mirror those constants for inspection. It
does **not** claim Component Model runtime support. Such support requires an
upstream stable API and a separate design review.

## What we learned from wasmtime-go

We reviewed
[wasmtime-go at commit `539fc0ff`](https://github.com/bytecodealliance/wasmtime-go/tree/539fc0ffef31570486b7f5d5779ffc45b86baac7)
as a competing Go embedding API. The review is about ergonomics, contracts,
tests, and packaging. No wasmtime-go implementation code has been copied.

Several choices are especially effective:

- Its compact
  [`Engine` / `Store` / `Module` / `Instance` / `Linker` model](https://github.com/bytecodealliance/wasmtime-go/blob/539fc0ffef31570486b7f5d5779ffc45b86baac7/README.md)
  gives new users a short path to a running module.
- A reusable
  [`Linker`](https://github.com/bytecodealliance/wasmtime-go/blob/539fc0ffef31570486b7f5d5779ffc45b86baac7/linker.go)
  makes host definitions and WASI configuration composable instead of
  requiring per-instance boilerplate.
- It offers both explicit host functions and ergonomic
  [`WrapFunc`](https://github.com/bytecodealliance/wasmtime-go/blob/539fc0ffef31570486b7f5d5779ffc45b86baac7/func.go),
  and its `Store` data plus `Caller` model gives callbacks useful embedder
  state.
- Separate
  [`Error`](https://github.com/bytecodealliance/wasmtime-go/blob/539fc0ffef31570486b7f5d5779ffc45b86baac7/error.go)
  and
  [`Trap`](https://github.com/bytecodealliance/wasmtime-go/blob/539fc0ffef31570486b7f5d5779ffc45b86baac7/trap.go)
  types make execution failures inspectable.
- Fuel, epoch interruption, and store limits expose operational controls
  rather than treating execution as an unbounded black box; see its
  [`Store`](https://github.com/bytecodealliance/wasmtime-go/blob/539fc0ffef31570486b7f5d5779ffc45b86baac7/store.go).
- Its
  [runnable examples](https://github.com/bytecodealliance/wasmtime-go/blob/539fc0ffef31570486b7f5d5779ffc45b86baac7/example_test.go)
  and
  [platform CI](https://github.com/bytecodealliance/wasmtime-go/blob/539fc0ffef31570486b7f5d5779ffc45b86baac7/.github/workflows/main.yml)
  treat consumer experience as part of the product.

We apply those lessons independently:

| Observed strength | WasmEdge-go application | Deliberate opportunity to improve |
|---|---|---|
| Short default workflow | Keep `VM` as the one-object common path. | Preserve the explicit loader/validator/executor/store pipeline for embedders without forcing it on beginners. |
| Reusable host-definition layer | Make a configured host `Module` reusable across VM and explicit-pipeline registration. | Track transfer and registration ownership in Go so reuse cannot silently become double-free or use-after-free. |
| Explicit and reflective host functions | Keep both `HostFunc`/`NewFunction` and `WrapFunc`. | Expose invocation context through `CallContext`, preserve original Go errors, and keep panic containment mandatory. |
| Inspectable execution errors | Use `*wasmedge.Error` with `errors.Is`/`errors.As`. | Keep binding misuse sentinels separate and return standard context cancellation errors. |
| Runtime resource controls | Bind statistics, cost limits, interruptibility, and memory limits as normal configuration. | Test policy survival across native constructors and expose safe defaults rather than raw knobs alone. |
| Extensive examples and platform automation | Maintain executable examples plus minimum/current Go CI. | Add race, lifecycle, leak-debug, exact-runtime, ABI-relocation, and minimal external-consumer gates. |

There are also places where v2 intentionally chooses a different Go surface:

- wasmtime-go's
  [`Func.Call`](https://github.com/bytecodealliance/wasmtime-go/blob/539fc0ffef31570486b7f5d5779ffc45b86baac7/func.go)
  accepts and returns dynamically shaped `interface{}` values. WasmEdge-go
  uses `[]Value` so result arity and WASM kinds stay explicit and consistent.
- WasmEdge-go centers cancellation on `context.Context`, while still exposing
  the native async object for advanced use.
- Copying memory operations are the advertised default. The zero-copy slice
  is named `UnsafeSlice` and carries a narrow invalidation contract.
- The owned/borrowed/transferred taxonomy and the concurrency baseline above
  are part of the public design, rather than knowledge callers must infer
  from finalizers or FFI implementation details.
- WASI configuration should become capability-oriented and default-deny,
  rather than a convenience that implicitly mirrors the host process.

These are independent design responses to common embedding problems. We may
match useful behavior or Go conventions, but we do not reproduce wasmtime-go
source, internal data structures, callback registries, or FFI machinery.

## Stabilization gates

The design is ready for a stable v2 release only when:

1. the public stable WasmEdge 0.17.1 surface has the auditable accounting in
   [API_COVERAGE.md](API_COVERAGE.md);
2. lifecycle tests cover owned, borrowed, transferred, retained, and
   callback-scoped objects;
3. `go test -race`, debug-leak tests, and cancellation stress tests pass;
4. Linux CI proves the five corrected functions link to `WASMEDGE_0.17`;
5. minimum and current supported Go versions pass vet, tests, and an
   out-of-tree consumer build on Linux/macOS, with the configured Windows lane
   passing before a stable tag;
6. WASI defaults and every unsafe memory escape hatch are documented with
   runnable examples;
7. the migration guide distinguishes implemented APIs from planned
   conveniences; and
8. the Bazel system-SDK build passes against the same 0.17.1 release artifact.

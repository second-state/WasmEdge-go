# WasmEdge-go v2 Refactoring Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement remaining phases task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> **For interns:** read SPEC.md first, then Appendix A. Every `TODO(intern-*)` in the tree is a self-contained task with an acceptance test.

**Goal:** Replace the legacy WasmEdge-go binding with an idiomatic Go v2 module covering the stable WasmEdge 0.17 C API.

**Architecture:** Single root cgo package `wasmedge` (cgo types cannot cross packages), file-per-concept mirroring the upstream header split; two API layers (high-level `VM`, explicit `Loader/Validator/Executor/Store`); ownership-tracked `Close()` resources; one cgo trampoline per callback kind with `cgo.Handle` payloads.

**Tech Stack:** Go ≥ 1.24 (needs `runtime.AddCleanup`, `unsafe.Slice`, `runtime/cgo.Handle`), cgo against `libwasmedge` ≥ 0.17.0, GitHub Actions, golangci-lint.

## Global Constraints

- Module path is exactly `github.com/second-state/WasmEdge-go/v2`; root package name `wasmedge`.
- Minimum Go version: `go 1.24` in go.mod. No third-party runtime dependencies (test-only deps allowed but avoid; std lib preferred).
- Bind only stable headers; never call anything in `wasmedge_deprecated.h`. `wasmedge_experimental.h` is out of scope.
- Every public symbol has a doc comment; no `Get` prefixes; no `self` receivers; errors are `error`, never a custom result struct.
- Panics must never cross a cgo callback boundary (`recover()` in every trampoline).
- No `reflect.SliceHeader`; use `unsafe.Slice`/`unsafe.String`.
- All C-string/bytes helpers free in the same function via `defer` unless ownership is documented.
- Intern TODO tags: `TODO(intern-easy)`, `TODO(intern-medium)`, `TODO(intern-hard)` — each must state the C function(s) to bind, the pattern file to copy, and the test to extend.
- Local dev/test env (documented in CONTRIBUTING.md):
  `export WASMEDGE_DIR=$HOME/workspace/WasmEdge` (headers at `$WASMEDGE_DIR/include/api` + `$WASMEDGE_DIR/build/include/api`, lib at `$WASMEDGE_DIR/build/lib/api`).
- Commit style: conventional commits + `Signed-off-by: hydai <z54981220@gmail.com>`; run `lineguard` on touched files before each commit; run `go build ./... && go vet ./... && go test ./...` before each commit.

---

### Phase 0: Clear the ground

**Files:**
- Delete: `wasmedge/*.go`, `wasmedge/BUILD.bazel`, `main.go`, `BUILD.bazel` (root), `WORKSPACE`, `MODULE.bazel.lock`
- Keep: `LICENSE`, `Changelog.md`, `README.md` (rewritten in Phase 8), `MODULE.bazel` (content emptied to a stub + TODO, Bazel regen tracked in Phase 9), `.github/workflows/` (rewritten in Phase 8)
- Create: `go.mod` (v2), `doc.go`, `SPEC.md`, `PLAN.md` (this file)

**Steps:**
- [x] Write SPEC.md + PLAN.md, commit as `docs: add v2 redesign spec and refactoring plan`
- [ ] `git rm -r wasmedge main.go BUILD.bazel WORKSPACE` (keep MODULE.bazel stub)
- [ ] `go.mod`: `module github.com/second-state/WasmEdge-go/v2`, `go 1.24`
- [ ] `doc.go`: package doc with a 20-line usage example
- [ ] Verify: `go build ./...` (compiles empty package), commit `feat!: start v2 module, remove legacy binding`

### Phase 1: cgo bridge + primitives

**Files:**
- Create: `cgo.go` (build flags: default `-lwasmedge`; document `CGO_CFLAGS`/`CGO_LDFLAGS` overrides for local builds)
- Create: `cstring.go` — internal `WasmEdge_String`/`WasmEdge_Bytes` ⇄ Go conversions:
  `toWasmEdgeString(s string) C.WasmEdge_String` (owned; caller defers delete), `wrapBytes([]byte)` (borrow, zero-copy + KeepAlive), `goString(C.WasmEdge_String) string`, `goStringN(names []C.WasmEdge_String) []string`
- Create: `errors.go` — `ErrCategory`, `ErrCode` (+ seeded constants), `Error`, `Terminate` sentinel, `newResult(C.WasmEdge_Result) error`, `toResult(error) C.WasmEdge_Result`
- Create: `wasmedge.go` — `Version()`, `VersionMajor/Minor/Patch()`, init version gate
- Create: `log.go` — `SetLogLevel`, `SetLogOff`, slog bridge + log-callback trampoline
- Test: `errors_test.go`, `wasmedge_test.go`

**Interfaces (produced, relied on by every later phase):**
- `newResult(res C.WasmEdge_Result) error` — nil on OK (incl. Terminate)
- `toResult(err error) C.WasmEdge_Result` — inverse, for host functions
- `toWasmEdgeString/goString`, `wrapBytes`
- `type Error struct{ Category ErrCategory; Code ErrCode; Message string }`

**Steps:** write tests (version round-trip, error mapping incl. Terminate-is-nil, Error.Is) → implement → `go test ./...` PASS → commit `feat: add cgo bridge, error mapping, version gate, logging`

### Phase 2: values and types

**Files:**
- Create: `valtype.go` (`ValType`, `ValKind`, constructors/predicates), `value.go` (`Value`, constructors `I32/I64/F32/F64/V128/FuncRefValue/ExternRefValue/NullRef`, accessors, `packValues/unpackValues` helpers), `externref.go` (cgo.Handle pinning)
- Create: `limits.go` (plain struct → C LimitContext materializer), `types.go` (`FunctionType`, `TableType`, `MemoryType`, `GlobalType`, `TagType`, `ImportType`, `ExportType`)
- Create: `resource.go` — ownership helper (owned/borrowed/transferred + AddCleanup arm/disarm)
- Test: `value_test.go`, `types_test.go`

**Interfaces:**
- `func packValues([]Value) (*C.WasmEdge_Value, C.uint32_t, func())`, `func unpackValues(ptr *C.WasmEdge_Value, n C.uint32_t) []Value`
- `resource.close(func())`, `resource.markTransferred()`, `newOwned/newBorrowed`
- `NewFunctionType(params, results []ValType) *FunctionType`

**Steps:** tests (value round-trip per kind incl. V128 lanes, externref pin/unpin, double-Close idempotency, limits equality) → implement → PASS → commit `feat: add typed values, value types, ownership tracking`

### Phase 3: execution pipeline

**Files:**
- Create: `config.go` (declarative `Config` + `build()`), `statistics.go`, `loader.go`, `validator.go`, `executor.go`, `store.go`, `astmodule.go`
- Test: `pipeline_test.go` (loader→validator→executor→invoke over fixture), `config_test.go`

**Interfaces:**
- `NewLoader(*Config) (*Loader, error)`, `(*Loader).LoadBytes([]byte) (*ASTModule, error)`, `LoadFile`, `Serialize(*ASTModule) ([]byte, error)`
- `NewValidator(*Config)`, `(*Validator).Validate(*ASTModule) error`
- `NewExecutor(*Config, ...ExecutorOption) (*Executor, error)` with `WithStats(*Statistics)`
- `(*Executor).Instantiate(*Store, *ASTModule) (*Module, error)`, `Register`, `RegisterImport`, `RegisterImportWithAlias`, `Invoke(*Function, ...Value) ([]Value, error)`, `InvokeContext`
- `NewStore() *Store`, `(*Store).Module(name) (*Module /*borrowed*/, bool)`, `ModuleNames() []string`

**Steps:** fixture-driven TDD as above → commit `feat: add config and loader/validator/executor/store pipeline`

### Phase 4: instances + host functions (the hard core — NOT intern work)

**Files:**
- Create: `module.go`, `function.go` (HostFunc, trampoline `//export wasmedgego_hostFuncInvoke`, `NewFunction`, `WrapFunc` reflection), `callcontext.go`, `memory.go`, `table.go`, `global.go`, `tag.go`, `hostdata.go` (module host-data finalizer trampoline)
- Create: `internal/testwasm/testwasm.go` (hand-encoded fixtures: Add, Fib, HostCall, Memory, InfiniteLoop)
- Test: `hostfunc_test.go` (round-trip, panic containment, Terminate, externref passing through a host fn), `memory_test.go`, `module_test.go`

**Steps:** TDD as above → commit `feat: add module/memory/table/global instances and host functions`

### Phase 5: VM + async

**Files:**
- Create: `vm.go`, `async.go` (`Execution`, `ExecuteContext` cancellation)
- Test: `vm_test.go` (RunBytes happy path, staged pipeline, ExecuteContext cancel on InfiniteLoop fixture with `-race`)

**Steps:** TDD → commit `feat: add VM and context-aware async execution`

### Phase 6: WASI + plugins

**Files:**
- Create: `wasi.go`, `plugin.go`
- Test: `wasi_test.go` (exit code via proc_exit fixture — fixture added here)

**Steps:** TDD → commit `feat: add WASI module and plugin loading`

### Phase 7: compiler + tools

**Files:**
- Create: `compiler.go`, `tools.go`
- Test: `compiler_test.go` (skips when AOT unavailable in lib build)

**Steps:** TDD → commit `feat: add AOT compiler and driver entry points`

### Phase 8: docs, examples, CI

**Files:**
- Create: `example_test.go` (pkg.go.dev runnable examples), `CONTRIBUTING.md` (build env, intern guide), rewrite `README.md`, `docs/MIGRATION.md`
- Create: `.golangci.yml`, rewrite `.github/workflows/ci.yml` (matrix ubuntu-24.04/macos-14, install WasmEdge 0.17.0, vet+lint+test -race), drop stale workflows
- Steps: `go vet ./... && go test -race ./...` PASS → commit `docs+ci: v2 documentation, examples, lint, CI matrix`

### Phase 9: deferred follow-ups (tracked, not in this branch)

- [ ] File upstream WasmEdge bug: `WasmEdge_LoaderSerializeASTModule` aborts
  with `std::system_error: mutex lock failed` on 0.17.0-168-gad9d34498
  (pure-C reproducer confirmed; see the comment on TestSerializeRoundTrip in
  pipeline_test.go). Re-enable that test via `WASMEDGE_TEST_SERIALIZE=1`
  after the fix.
- [ ] Regenerate Bazel BUILD files for root-package layout (gazelle) — restores #58
- [ ] Windows CI lane (cgo flags exist; needs runner validation)
- [ ] `wasmedge_experimental.h` behind `//go:build wasmedge_experimental`
- [ ] Typed plugin sub-packages (wasi_nn) if demand appears
- [ ] Burn down Appendix A intern tasks; tag `v2.0.0-alpha.1`

---

## Appendix A: Intern Handoff Tasks

Ground rules: one task = one PR; copy the referenced pattern file; extend the
referenced test; run `go vet ./... && go test ./...`; conventional commit +
sign-off. Difficulty: easy ≈ half a day, medium ≈ 1–2 days.

| # | Tag in tree | File | Task | Pattern to copy | Test to extend |
|---|---|---|---|---|---|
| A1 | `TODO(intern-easy)` | `errors.go` | Complete `ErrCode` constants + `String()` from `enum.inc` `UseErrCode` table | seeded constants above the TODO | `errors_test.go` |
| A2 | `TODO(intern-easy)` | `statistics.go` | Bind `SetCostTable`, `SetCostLimit`, `Clear` | `InstrCount` in same file | `pipeline_test.go/TestStatistics` |
| A3 | `TODO(intern-easy)` | `types.go` | `ImportType/ExportType` accessors for Table/Memory/Tag/Global | the Function accessors in same file | `types_test.go` |
| A4 | `TODO(intern-easy)` | `vm.go` | `RunFile`, `ExecuteRegistered`, `RegisterModuleFromFile/FromImportWithAlias` | `RunBytes`/`RegisterModule` in same file | `vm_test.go` |
| A5 | `TODO(intern-easy)` | `plugin.go` | `(*Plugin).ModuleNames`, `InitWASINN` | `PluginNames` in same file | `plugin_test.go` (new) |
| A6 | `TODO(intern-easy)` | `table.go` | `Grow`, `Size`, `TableType` accessor | `memory.go` Grow/PageCount | `module_test.go/TestTable` |
| A7 | `TODO(intern-medium)` | `global.go`, `tag.go` | Global set/get incl. v128 + Tag type accessors | `table.go` Get/Set | `module_test.go/TestGlobal` |
| A8 | `TODO(intern-medium)` | `function.go` | Extend `WrapFunc` kinds: `[16]byte` (v128), `*ExternRef`, `*Function` params/results | existing i32/i64/f32/f64 switch arms | `hostfunc_test.go/TestWrapFuncKinds` |
| A9 | `TODO(intern-medium)` | `wasi.go` | `NativeHandler`, `InitWASI` re-init on borrowed VM module | `ExitCode` in same file | `wasi_test.go` |
| A10 | `TODO(intern-medium)` | `async.go` | `WaitFor(d time.Duration) bool` + `Execution` introspection | `Wait` in same file | `vm_test.go/TestExecuteAsync` |
| A11 | `TODO(intern-medium)` | `log.go` | Map engine log levels→slog levels incl. Trace/Critical edge cases; attach LoggerName/ThreadId attrs | callback trampoline above the TODO | `log_test.go` (new) |
| A12 | `TODO(intern-medium)` | `tools.go` | `DriverWasiNNRPCServer` + argv UTF-8 helpers on Windows | `DriverTool` in same file | manual (documented in file) |
| A13 | `TODO(intern-easy)` | `config_string.go` | `String()` for `Proposal`, `Standard`, `RunMode`, `HostRegistration` | `ValKind.String()` in `valtype.go` | `config_test.go` |
| A14 | `TODO(intern-hard)` | `memory.go` | Shared-memory (Threads) doc pass + aliasing tests; decide `UnsafeSlice` policy under growth | doc block on `UnsafeSlice` | `memory_test.go/TestSharedMemory` |

## Appendix B: Verification Matrix

| Check | Command | Gate |
|---|---|---|
| Build | `go build ./...` | every commit |
| Vet | `go vet ./...` | every commit |
| Unit + race | `go test -race ./...` | every commit |
| Lint | `golangci-lint run` | CI |
| Leak debug | `go test -tags wasmedge_debug ./...` | CI (informational) |
| Version gate | `go test -run TestVersion ./...` against 0.17.x lib | CI |

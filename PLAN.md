# WasmEdge-go v2 Refactoring Plan

> This is the historical execution plan for the v2 rewrite. Read SPEC.md and
> docs/DESIGN.md for the current API contract; Appendix A records completed
> work and the remaining platform-gated item.

**Goal:** Replace the legacy WasmEdge-go binding with an idiomatic Go v2
module covering the stable WasmEdge 0.17 C API. The final declaration
accounting is in `docs/API_COVERAGE.md`.

**Architecture:** Single root cgo package `wasmedge` (cgo types cannot cross packages), file-per-concept mirroring the upstream header split; two API layers (high-level `VM`, explicit `Loader/Validator/Executor/Store`); plain-Go type descriptors plus ownership-tracked native wrappers; explicit-only `Close()` teardown; one cgo trampoline per callback kind.

**Tech Stack:** Active supported Go releases (currently 1.25 and 1.26; needs `runtime.AddCleanup`, `unsafe.Slice`, `runtime/cgo.Handle`), cgo against `libwasmedge` ≥ 0.17.1 and < 0.18.0, GitHub Actions, golangci-lint.

## Global Constraints

- Module path is exactly `github.com/second-state/WasmEdge-go/v2`; root package name `wasmedge`.
- Supported Go versions follow the active release lines: `go 1.25` in go.mod,
  with CI on Go 1.25 and 1.26. No third-party runtime dependencies (test-only
  deps allowed but avoid; std lib preferred).
- Bind only stable headers; never call anything in `wasmedge_deprecated.h`. `wasmedge_experimental.h` is out of scope.
- Every public symbol has a doc comment; no `Get` prefixes; no `self` receivers; errors are `error`, never a custom result struct.
- Panics must never cross a cgo callback boundary (`recover()` in every trampoline).
- No `reflect.SliceHeader`; use `unsafe.Slice`/`unsafe.String`.
- All C-string/bytes helpers free in the same function via `defer` unless ownership is documented.
- Remaining TODOs must state their upstream/build constraint and required
  verification environment.
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
- [x] `git rm -r wasmedge main.go BUILD.bazel WORKSPACE` (keep MODULE.bazel stub)
- [x] `go.mod`: `module github.com/second-state/WasmEdge-go/v2`, `go 1.25`
- [x] `doc.go`: package doc with a usage example
- [x] Verify the root-package v2 module builds after removing the legacy tree

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
- Create: `resource.go` — ownership helper (owned/borrowed/transferred,
  dependency leases, VM generations, and leak-only AddCleanup diagnostics)
- Test: `value_test.go`, `types_test.go`

**Public interfaces (final shape):**
- Plain values: `FunctionType{Params, Results}`, `TableType{Element, Limits}`,
  `MemoryType{Limits}`, `GlobalType{Value, Mutability}`, and
  `TagType{Signature}`. There are no `New*Type` constructors or type `Close`
  methods.
- Native wrappers use owned/borrowed/transferred lifetime state. GC cleanup
  never invokes C; `-tags wasmedge_debug` reports an omitted explicit Close.

**Steps:** tests (value round-trip per kind incl. V128 lanes, externref pin/unpin, double-Close idempotency, limits equality) → implement → PASS → commit `feat: add typed values, value types, ownership tracking`

### Phase 3: execution pipeline

**Files:**
- Create: `config.go` (declarative `Config` + `build()`), `statistics.go`, `loader.go`, `validator.go`, `executor.go`, `store.go`, `astmodule.go`
- Test: `pipeline_test.go` (loader→validator→executor→invoke over fixture), `config_test.go`

**Interfaces:**
- `NewLoader(*Config) (*Loader, error)`, `(*Loader).LoadBytes([]byte) (*ASTModule, error)`, `LoadFile`, `Serialize(*ASTModule) ([]byte, error)`
- `NewValidator(*Config) (*Validator, error)`, `(*Validator).Validate(*ASTModule) error`
- `NewExecutor(*Config, ...ExecutorOption) (*Executor, error)` with `WithStats(*Statistics)`
- `(*Executor).Instantiate(*Store, *ASTModule) (*Module, error)`, `Register`, `RegisterImport`, `RegisterImportWithAlias`, `Invoke(*Function, ...Value) ([]Value, error)`, `InvokeAsync`, `InvokeContext`
- `NewStore() *Store`, `(*Store).Module(name) (*Module /*borrowed*/, bool)`, `ModuleNames() []string`

**Steps:** fixture-driven TDD as above → commit `feat: add config and loader/validator/executor/store pipeline`

### Phase 4: instances + host functions (the hard core — NOT intern work)

**Files:**
- Create: `module.go`, `function.go` (HostFunc, trampoline `//export wasmedgego_hostFuncInvoke`, `NewFunction`, `WrapFunc` reflection), `callcontext.go`, `memory.go`, `table.go`, `global.go`, `tag.go`, `hostdata.go` (module host-data finalizer trampoline)
- Create: `internal/testwasm/testwasm.go` (hand-encoded fixtures: Add, Fib, HostCall, Memory, InfiniteLoop)
- Test: `hostfunc_test.go` (round-trip, panic containment, Terminate, externref passing through a host fn), `memory_test.go`, `module_test.go`

**Steps:** TDD as above → commit `feat: add module/memory/table/global instances and host functions`

**Final constructor shape:** `NewFunction`, `NewTable`,
`NewTableWithInit`, `NewMemory`, and `NewGlobal` consume plain descriptors
and return `(*T, error)`. Allocation-only `NewModule` is semantically
infallible and panics only on impossible native allocation failure.

### Phase 5: VM + async

**Files:**
- Create: `vm.go`, `async.go` (`Execution`, `ExecuteContext` cancellation)
- Test: `vm_test.go` (RunBytes happy path, staged pipeline, ExecuteContext cancel on InfiniteLoop fixture with `-race`)

**Steps:** TDD → commit `feat: add VM and context-aware async execution`

**Final lifetime shape:** one active execution per VM/Executor; exactly one
terminal `Execution.Wait` or `Execution.Close`; observational `WaitFor`;
request-only `Cancel`; explicit `ErrCancellationRace`/`ErrUnusable` handling
for WasmEdge 0.17.1's shared stop token. Active and registered module views
carry generation guards, and `VM.Reset() error` refuses leased/in-flight
state before invalidating both generations.

### Phase 6: WASI + plugins

**Files:**
- Create: `wasi.go`, `plugin.go`
- Test: `wasi_test.go` (exit code via proc_exit fixture — fixture added here)

**Steps:** TDD → commit `feat: add WASI module and plugin loading`

`NewWASIModule(WASIConfig) (*Module, error)` requires an explicit
`WASIStdio` policy. VM registration alone leaves the fd table empty;
`DiscardWASIStdio` supplies binding-owned null streams,
`InheritWASIStdio` grants process streams, and `RedirectWASIStdio` grants a
complete set of three caller-owned files. Unix descriptors must fit `int32`;
redirection on Windows is rejected because `os.File.Fd` is a HANDLE, not
WasmEdge's expected CRT descriptor. The module retains caller-owned stdio
wrappers and binding-owned discard descriptors while they remain native
dependencies, and WASI-specific calls require private constructor/accessor
provenance rather than matching a module name.
Because WasmEdge 0.17.1 cannot replace initialized fd-table entries, re-init
accepts args/env changes only when preopens, policy, and files are identical;
mapping changes return `ErrWASIResourceMappingImmutable`.

### Phase 7: compiler + tools

**Files:**
- Create: `compiler.go`, `tools.go`
- Test: `compiler_test.go` (skips when AOT unavailable in lib build)

**Steps:** TDD → commit `feat: add AOT compiler and driver entry points`

### Phase 8: docs, examples, CI

**Files:**
- Create: `example_test.go` (pkg.go.dev runnable examples), `CONTRIBUTING.md` (build env, intern guide), rewrite `README.md`, `docs/MIGRATION.md`
- Create: `.golangci.yml`, rewrite `.github/workflows/ci.yml` (matrix ubuntu-24.04/macos-15 plus Windows 2025, install WasmEdge 0.17.1, vet+lint+test -race), drop stale workflows
- Steps: `go vet ./... && go test -race ./...` PASS → commit `docs+ci: v2 documentation, examples, lint, CI matrix`

### Phase 9: follow-ups and platform constraints

- [x] Guard the upstream `WasmEdge_LoaderSerializeASTModule` process abort on
  the official 0.17.1 `darwin/arm64` artifact. `Loader.Serialize` returns
  `ErrSerializeUnsupported` before native entry on that platform; Linux,
  Windows, and other architectures continue to run the native round-trip
  test. Remove the targeted guard after an upstream fix. Filing an upstream
  issue remains an external follow-up.
- [x] Restore Bazel 9.2.0/rules_go 0.62.0 BUILD files for the root-package
  layout using a caller-provided `WASMEDGE_SDK` prefix.
- [ ] When bumping to the next C API line: `WasmEdge_ModuleInstanceAdd*`
  return `WasmEdge_Result` at upstream HEAD (void in released 0.17.x) —
  switch module.go's Add* bodies to `newResult(...)`; the Go signatures
  already return error so this is non-breaking. Upstream HEAD also removed
  the post-0.17 MaxGC configure knobs; do not bind them.
- [x] Configure checksum-pinned Ubuntu 24.04, macOS 15, and Windows 2025 CI
  lanes for Go 1.25 and 1.26, including a Bazel system-SDK test on each Go
  1.26 host lane. The refreshed matrix still needs an exact-commit hosted run
  before a stable tag.
- [x] Make WASI stdio a required explicit capability policy. VM registration
  starts with an empty fd table; discard, inherit, and redirect are explicit
  initialization paths, and the zero value is rejected before native entry.
- [x] Make the 0.17.1 C API audit reproducible with a per-symbol manifest and
  a verifier that reparses the official SDK headers and checks all direct
  production references in CI.
- [ ] `wasmedge_experimental.h` behind `//go:build wasmedge_experimental`
- [ ] Typed plugin sub-packages (wasi_nn) if demand appears
- [x] Explicitly defer Appendix A's platform-gated A12: the official 0.17.1
  SDKs do not export it, so enabling it requires a feature-matched custom SDK
  and dedicated build tag/CI lane. Tagging `v2.0.0-alpha.1` remains a
  separate release action after the exact-commit hosted matrix passes.

---

## Appendix A: Remaining Handoff Tasks

Completed entries remain in this table so the implementation history is
visible. Ground rules for open work: one task = one PR; extend the referenced
test; run `go vet ./... && go test ./...`; use a Conventional Commit subject.

| # | Status | File | Task | Test |
|---|---|---|---|---|
| A1 | done | `errors.go` | Complete all 160 WasmEdge 0.17.1 `ErrCode` constants and `String()` values | `errors_test.go` |
| A2 | done | `statistics.go` | Bind cost table, cost limit, and clear; preserve policy across Executor creation | `pipeline_test.go/TestStatistics` |
| A3 | done | `astmodule.go` | Typed import/export accessors for Table, Memory, Tag, and Global | `types_test.go` |
| A4 | done | `vm.go` | File registration/run, alias imports, registered sync/async/context execution | `vm_test.go` |
| A5 | done | `plugin.go` | `(*Plugin).ModuleNames`, `InitWASINN` | `plugin_test.go` |
| A6 | done | `table.go` | `Grow`, `Size`, `Type`, and reference lifetime checks | `table_test.go` |
| A7 | done | `global.go`, `tag.go` | Global get/set including v128; Tag type accessors | `global_test.go` |
| A8 | done | `function.go` | `WrapFunc` support for v128, externref, and funcref; canonical result-root cycle rejection | `hostfunc_test.go`, `reference_cycle_test.go` |
| A9 | done | `wasi.go` | Provenance-checked exit/native-handler APIs; args/env re-init with immutable 0.17.1 stdio/preopen mappings; stdio rooting on standalone/borrowed VM modules | `wasi_test.go`, platform-specific stdio tests |
| A10 | done | `async.go`, `vm.go`, `resource.go` | Terminal async ownership, cancellation-race poisoning, transactional invocation roots, and generation-safe result/view lifetimes | `vm_test.go`, `invocation_test.go` |
| A11 | done | `log.go` | Complete Trace/Critical slog mapping and structured attributes | `log_test.go` |
| A12 | explicitly deferred | `tools.go` | WASI-NN RPC driver is absent from official 0.17.1 SDK builds; future support requires a build tag, feature-matched runtime, and CI | platform/manual |
| A13 | done | `config_string.go` | `String()` for configuration enums | `config_test.go` |
| A14 | done | `memory.go` | Reject shared-memory Go slices; document aliasing contract | `memory_test.go` |
| A15 | done, hosted validation pending | `tools_console_windows.go`, `.github/workflows/ci.yml` | Configure Windows console output as UTF-8, split platform-specific WASI stdio tests, and add Go 1.25/1.26 Windows CI lanes | exact-commit hosted matrix |

## Appendix B: Verification Matrix

| Check | Command | Gate |
|---|---|---|
| Build | `go build ./...` | every commit |
| Vet | `go vet ./...` | every commit |
| Unit + race | `go test -race ./...` | every commit |
| Strict cgo pointers | `GOEXPERIMENT=cgocheck2 go test ./...` | CI |
| Lint | `golangci-lint run` | CI |
| Leak debug | `go test -tags wasmedge_debug ./...` | CI (informational) |
| Version gate | `go test -run TestVersion ./...` against a >= 0.17.1, < 0.18 lib | CI |
| Bazel | `bazel test --repo_env=WASMEDGE_SDK=/absolute/sdk/prefix //...` | release candidate |
| Windows | checksum-pinned `windows-2025`, Go 1.25 and 1.26 | required before stable |

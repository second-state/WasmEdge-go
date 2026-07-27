# WasmEdge-go

Go bindings for the [WasmEdge](https://github.com/WasmEdge/WasmEdge) runtime.

> **v2 status: alpha candidate; the refreshed hosted matrix awaits an
> exact-commit run.** This branch is the ground-up redesign for the WasmEdge
> 0.17.1 C API.
> Its design principles are in [docs/DESIGN.md](docs/DESIGN.md), the current
> API in [SPEC.md](SPEC.md), the auditable C API accounting in
> [docs/API_COVERAGE.md](docs/API_COVERAGE.md), the implementation history and
> remaining platform work in [PLAN.md](PLAN.md), and common v1 → v2 migration
> paths in [docs/MIGRATION.md](docs/MIGRATION.md).
> v1 (`github.com/second-state/WasmEdge-go/wasmedge`) remains importable via
> Go module versioning.

## Requirements

- Go 1.25 or 1.26 (the active supported Go release lines)
- WasmEdge shared library ≥ 0.17.1 and < 0.18.0

```bash
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/0.17.1/utils/install.sh \
  | bash -s -- -v 0.17.1
source $HOME/.wasmedge/env
```

On Linux and Android, rebuild the Go binary after upgrading from WasmEdge
0.17.0 to 0.17.1. The 0.17.1 release added ELF symbol versions for the five
Limit-related functions whose ABI changed in 0.17; reusing an object linked
against 0.17.0 is unsafe.

## Quick start

```go
package main

import (
	"fmt"
	"log"

	wasmedge "github.com/second-state/WasmEdge-go/v2"
)

func main() {
	vm, err := wasmedge.NewVM(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer vm.Close()

	out, err := vm.RunBytes(wasmBytes, "fib", wasmedge.I32(21))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out[0].I32()) // 17711
}
```

Host functions are plain Go functions:

```go
add, err := wasmedge.WrapFunc(func(a, b int32) int32 { return a + b })
if err != nil {
	return err
}
defer add.Close() // closes on AddFunction failure; inert after transfer

env := wasmedge.NewModule("env")
defer func() {
	_ = vm.Close()  // releases the registered-import lease
	_ = env.Close() // caller still owns the module
}()
if err := env.AddFunction("add", add); err != nil {
	return err
}
if err := vm.RegisterImport(env); err != nil {
	return err
}
```

`RegisterImport` keeps the module caller-owned. At shutdown, close or reset
the VM before closing `env`; while registered, `env.Close` reports
`ErrInUse`.

One upstream lifetime detail matters with `WithExternalStore`: `VM.Reset`
maps to `WasmEdge_VMCleanup`, which clears every non-built-in registration in
that Store, including modules registered before the VM was created. `VM.Close`
destroys VM-owned guest registrations, but preserves both pre-existing
registrations and caller-owned imports/aliases. Those imports remain direct
Store registrations until their module is closed.

WASM type descriptors are plain Go values. Constructors that validate a
descriptor or initializer return an error:

```go
memory, err := wasmedge.NewMemory(wasmedge.MemoryType{
	Limits: wasmedge.Limits{Min: 1, HasMax: true, Max: 16},
})
if err != nil {
	return err
}
defer memory.Close()
```

Cancellation and timeouts use `context.Context`:

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
out, err := vm.ExecuteContext(ctx, "main") // interrupts runaway WASM
```

Host functions reached through a context-aware invocation observe that same
context through `call.Context()` and should select on its `Done` channel when
performing blocking Go work.

Context methods always wait for the native operation to settle before
returning. The lower-level `Execution` API, created by `VM.ExecuteAsync` or
`Executor.InvokeAsync`, requires exactly one terminal `Wait` or `Close`;
`WaitFor` only observes and `Cancel` only requests interruption. WasmEdge
0.17.1 has a rare shared-token completion race: it is reported as
`ErrCancellationRace` by the manual terminal API. A context method returns the
context's error after draining, but still marks the originating VM or Executor
`ErrUnusable`; recreate it before another execution.

More runnable examples live in [example_test.go](example_test.go) and on
pkg.go.dev.

## What changed from v1 (highlights)

- Runtime operations return `error`; engine failures are typed
  `*wasmedge.Error` values (`errors.Is`/`errors.As` work).
- `Release()` is gone: resources are `io.Closer`s with idempotent `Close`,
  ownership transfer and dependency leases. Native teardown is
  explicit-only: garbage collection never substitutes for `Close`; build
  with `-tags wasmedge_debug` to report leaked wrappers.
- WASM values are typed `wasmedge.Value`s (`I32(…)`, `v.I32()`), not
  `interface{}`.
- Configuration, limits, and function/table/memory/global/tag type
  descriptors are plain Go values with no lifetime to manage.
- `Config.Effective()` resolves native defaults and overrides into ordinary Go
  data. A nil receiver reports WasmEdge defaults; in 0.17.1 those compiler
  defaults are O3 and universal Wasm output.
- Standalone WASI construction returns `(*Module, error)` and validates
  its explicit `WASIStdio` policy before entering the runtime. VM
  registration alone leaves the fd table empty; `DiscardWASIStdio()` gives
  stdin EOF and output sinks without ambient process streams,
  `InheritWASIStdio()` deliberately grants the process streams, and
  `RedirectWASIStdio(in, out, errOut)` grants three caller-owned files.
  WASI-only methods validate private provenance; redirected `*os.File`s stay
  caller-owned but are kept reachable until module/generation teardown.
  WasmEdge 0.17.1 fixes the stdio/preopen mapping at first initialization;
  later initialization may update args/env only when those mappings are
  unchanged.
- Host functions: one reflective line (`WrapFunc`) or the explicit
  `HostFunc` form; Go panics never cross into the engine.
- `log/slog` integration for engine logs.

See [docs/MIGRATION.md](docs/MIGRATION.md) for common migration paths and
intentionally removed API families.

## Building against a local WasmEdge checkout

```bash
export WASMEDGE_DIR=$HOME/workspace/WasmEdge
export CGO_CFLAGS="-I$WASMEDGE_DIR/build/include/api"
export CGO_LDFLAGS="-L$WASMEDGE_DIR/build/lib/api -Wl,-rpath,$WASMEDGE_DIR/build/lib/api -lwasmedge"
go test ./...
```

## Bazel

The Bazel build is pinned to Bazel 9.2.0, rules_go 0.62.0, and Go 1.26.5.
It deliberately does not download or select a WasmEdge runtime. Point it at a
standard WasmEdge 0.17.1 SDK prefix:

```text
include/wasmedge/wasmedge.h
lib/libwasmedge.so.0                 # Linux
lib/libwasmedge.0.dylib              # macOS
lib/wasmedge.lib + bin/wasmedge.dll  # Windows
```

Then build and test with:

```bash
bazel test --repo_env=WASMEDGE_SDK=/absolute/path/to/sdk //...
```

The repository rule selects the host platform's SDK layout; Bazel
cross-compilation and remote execution with a different target platform are
not currently supported. CI is configured to exercise the native Bazel path
on Linux, macOS, and Windows; the refreshed workflow still awaits its first
exact-commit hosted validation.

## Known upstream/platform constraints

- The official WasmEdge 0.17.1 `darwin/arm64` artifact can abort the process
  inside `Loader.Serialize`; the binding prevents native entry on that
  platform and returns `ErrSerializeUnsupported`, which wraps
  `errors.ErrUnsupported`. Linux, Windows, and other architectures continue
  to test native serialization.
- The official 0.17.1 SDK does not export the build-conditional WASI-NN RPC
  driver. See [the API coverage decision](docs/API_COVERAGE.md#wasi-nn-rpc-driver).

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development guide and remaining
platform validation.

## License

Apache 2.0 — see [LICENSE](LICENSE).

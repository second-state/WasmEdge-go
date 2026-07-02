# WasmEdge-go

Go bindings for the [WasmEdge](https://github.com/WasmEdge/WasmEdge) runtime.

> **v2 status: scaffolding complete, pre-alpha.** This branch is the ground-up
> redesign for the WasmEdge 0.17 C API. The design is in [SPEC.md](SPEC.md),
> the remaining work (including mentored starter tasks) in [PLAN.md](PLAN.md),
> and the v1 → v2 symbol mapping in [docs/MIGRATION.md](docs/MIGRATION.md).
> v1 (`github.com/second-state/WasmEdge-go/wasmedge`) remains importable via
> Go module versioning.

## Requirements

- Go ≥ 1.24
- WasmEdge shared library ≥ 0.17.0

```bash
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh \
  | bash -s -- -v 0.17.0
source $HOME/.wasmedge/env
```

## Quick start

```go
package main

import (
	"fmt"
	"log"

	wasmedge "github.com/second-state/WasmEdge-go/v2"
)

func main() {
	vm, err := wasmedge.NewVM(&wasmedge.Config{WASI: true})
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
env := wasmedge.NewModule("env")
defer env.Close()
env.AddFunction("add", wasmedge.MustWrapFunc(func(a, b int32) int32 { return a + b }))
vm.RegisterImport(env)
```

Cancellation and timeouts use `context.Context`:

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
out, err := vm.ExecuteContext(ctx, "main") // interrupts runaway WASM
```

More runnable examples live in [example_test.go](example_test.go) and on
pkg.go.dev.

## What changed from v1 (highlights)

- Every fallible call returns `error`; engine failures are typed
  `*wasmedge.Error` values (`errors.Is`/`errors.As` work).
- `Release()` is gone: resources are `io.Closer`s with idempotent `Close`,
  ownership transfer tracking, and a GC safety net (leak reports under
  `-tags wasmedge_debug`).
- WASM values are typed `wasmedge.Value`s (`I32(…)`, `v.I32()`), not
  `interface{}`.
- Configuration is a declarative struct with no lifetime to manage.
- Host functions: one reflective line (`WrapFunc`) or the explicit
  `HostFunc` form; Go panics never cross into the engine.
- `log/slog` integration for engine logs.

See [docs/MIGRATION.md](docs/MIGRATION.md) for the full table.

## Building against a local WasmEdge checkout

```bash
export WASMEDGE_DIR=$HOME/workspace/WasmEdge
export CGO_CFLAGS="-I$WASMEDGE_DIR/build/include/api"
export CGO_LDFLAGS="-L$WASMEDGE_DIR/build/lib/api -Wl,-rpath,$WASMEDGE_DIR/build/lib/api -lwasmedge"
go test ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the development guide and the
mentored starter-task list.

## License

Apache 2.0 — see [LICENSE](LICENSE).

# Contributing to WasmEdge-go

## Development environment

Use Go 1.25 or 1.26. The project follows Go's active-release policy rather
than carrying older toolchains after upstream support ends; the workflow is
configured to test both active lines.

Install WasmEdge 0.17.1 (or a later 0.17.x patch release):

```bash
# Release install
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/0.17.1/utils/install.sh \
  | bash -s -- -v 0.17.1
source $HOME/.wasmedge/env

# ...or point cgo at a local build tree
export WASMEDGE_DIR=$HOME/workspace/WasmEdge
export CGO_CFLAGS="-I$WASMEDGE_DIR/build/include/api"
export CGO_LDFLAGS="-L$WASMEDGE_DIR/build/lib/api -Wl,-rpath,$WASMEDGE_DIR/build/lib/api -lwasmedge"
```

Then the usual loop:

```bash
go build ./... && go vet ./... && go test ./... && go test -race ./...
go test -tags wasmedge_debug ./...   # reports wrappers leaked without Close
GOEXPERIMENT=cgocheck2 go test ./...
go run ./internal/apicoverage -sdk /absolute/path/to/wasmedge-0.17.1
```

The last command reparses the released stable headers and checks the exact
291-row classification manifest in `docs/wasmedge-0.17.1-api.csv`; run it
whenever a native binding or accepted WasmEdge C API line changes.

The Bazel path uses the standard release-SDK layout rather than cgo
environment flags. It pins Bazel 9.2.0 and Go 1.26.5, but intentionally leaves
the native runtime under caller control:

```bash
bazel test --repo_env=WASMEDGE_SDK=/absolute/path/to/wasmedge-0.17.1 //...
```

`WASMEDGE_SDK` must be absolute and contain
`include/wasmedge/wasmedge.h`. Linux and macOS require the corresponding
versioned library in `lib/`; Windows requires `lib/wasmedge.lib` and
`bin/wasmedge.dll`. The repository rule validates this before analysis.
It resolves the host OS, so cross-platform builds and remote execution for a
different target platform are not supported yet.

## Ground rules (enforced in review)

1. **Bind released headers only.** Nothing from `wasmedge_deprecated.h`;
   `wasmedge_experimental.h` is out of scope. When the build headers and
   upstream HEAD disagree, the released tag wins (see PLAN.md Phase 9 for a
   live example).
2. **KeepAlive discipline.** Every method that touches a wrapper's C
   pointer starts with `defer runtime.KeepAlive(x)`. This preserves callback
   state, reference roots, borrowed owners, and Go-backed arguments through
   the native call. See `resource.go` for the full ownership model
   (owned / borrowed / transferred).
3. **Panics never cross cgo callbacks.** Anything invoked from C recovers
   and converts to a result (see `callbacks.go`).
4. **No `Get` prefixes, no `self` receivers, doc comment on every exported
   symbol.** Errors are `error`; success is `nil`.
5. **Tests are fixture-driven.** WASM binaries are hand-assembled in
   `internal/testwasm` — no toolchains, no binary files, no network. Add new
   fixtures there with byte-level comments.
6. Run `go build ./... && go vet ./... && go test ./... &&
   go test -race ./...` before every commit;
   conventional commits with a `Signed-off-by` trailer.

## Remaining work

The implementation status and remaining platform-specific work are tracked in
**PLAN.md Appendix A**. Keep one focused concern per PR, add a regression test,
and update the table when finishing an entry.

Grep for your next task:

```bash
rg "TODO\\(intern-" --glob "*.go" --glob "*.bazel"
```

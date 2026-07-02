# Contributing to WasmEdge-go

## Development environment

Install WasmEdge 0.17.x (release install or a local build):

```bash
# Release install
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh \
  | bash -s -- -v 0.17.0
source $HOME/.wasmedge/env

# ...or point cgo at a local build tree
export WASMEDGE_DIR=$HOME/workspace/WasmEdge
export CGO_CFLAGS="-I$WASMEDGE_DIR/build/include/api"
export CGO_LDFLAGS="-L$WASMEDGE_DIR/build/lib/api -Wl,-rpath,$WASMEDGE_DIR/build/lib/api -lwasmedge"
```

Then the usual loop:

```bash
go build ./... && go vet ./... && go test -race ./...
go test -tags wasmedge_debug ./...   # reports wrappers leaked without Close
```

## Ground rules (enforced in review)

1. **Bind released headers only.** Nothing from `wasmedge_deprecated.h`;
   `wasmedge_experimental.h` is out of scope. When the build headers and
   upstream HEAD disagree, the released tag wins (see PLAN.md Phase 9 for a
   live example).
2. **KeepAlive discipline.** Every method that touches a wrapper's C
   pointer starts with `defer runtime.KeepAlive(x)`. Without it the GC may
   free the C object mid-call. See `resource.go` for the full ownership
   model (owned / borrowed / transferred).
3. **Panics never cross cgo callbacks.** Anything invoked from C recovers
   and converts to a result (see `callbacks.go`).
4. **No `Get` prefixes, no `self` receivers, doc comment on every exported
   symbol.** Errors are `error`; success is `nil`.
5. **Tests are fixture-driven.** WASM binaries are hand-assembled in
   `internal/testwasm` — no toolchains, no binary files, no network. Add new
   fixtures there with byte-level comments.
6. Run `go vet ./... && go test -race ./...` before every commit;
   conventional commits with a `Signed-off-by` trailer.

## Starter tasks (mentored)

Every `TODO(intern-easy|medium|hard)` in the tree is a self-contained task:
it names the C functions to bind, the pattern file to copy, and the test to
extend. The full list with acceptance criteria is **PLAN.md Appendix A**.
One task = one PR. Suggested first task: A1 (error-code table) or A2
(statistics setters).

Grep for your next task:

```bash
grep -rn "TODO(intern-" --include="*.go" --include="*.bazel" .
```

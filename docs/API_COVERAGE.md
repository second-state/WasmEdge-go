# WasmEdge 0.17.1 C API coverage

This document accounts for every exported declaration in the stable
[WasmEdge 0.17.1 C headers][headers]. It distinguishes direct native bindings
from deliberately idiomatic Go equivalents and from declarations that must
not be exposed by this runtime-consumer package.

The audit is pinned to the `0.17.1` tag. It excludes
`wasmedge_deprecated.h` and `wasmedge_experimental.h`, as required by the v2
contract.

The declaration-by-declaration source of truth is
[`wasmedge-0.17.1-api.csv`](wasmedge-0.17.1-api.csv). CI reparses the ten raw
headers from the checksum-verified official SDK, compares their exact symbol,
header, classification, and totals with that manifest, then verifies that
every `direct` entry has a production Go or native binding reference:

```bash
go run ./internal/apicoverage -sdk /absolute/path/to/wasmedge-0.17.1
```

## Accounting

The stable headers contain 290 runtime declarations across all platform and
feature guards. Of those, 272 are bound directly, 16 have an idiomatic
semantic equivalent, and 2 are documented omissions. The plugin header also
contains one provider-side export, bringing the textual export-declaration
total to 291:

| Header | Declarations | Direct | Semantic equivalent | Omitted | Provider non-goal |
|---|---:|---:|---:|---:|---:|
| `wasmedge_basic.h` | 23 | 17 | 6 | 0 | 0 |
| `wasmedge_value.h` | 32 | 32 | 0 | 0 | 0 |
| `wasmedge_configure.h` | 39 | 38 | 1 | 0 | 0 |
| `wasmedge_ast.h` | 47 | 47 | 0 | 0 | 0 |
| `wasmedge_instance.h` | 59 | 56 | 3 | 0 | 0 |
| `wasmedge_execution.h` | 30 | 30 | 0 | 0 | 0 |
| `wasmedge_vm.h` | 38 | 34 | 3 | 1 | 0 |
| `wasmedge_compiler.h` | 5 | 4 | 1 | 0 | 0 |
| `wasmedge_plugin.h` runtime API | 10 | 10 | 0 | 0 | 0 |
| `wasmedge_tools.h` | 7 | 4 | 2 | 1 | 0 |
| **Runtime total** | **290** | **272** | **16** | **2** | **0** |
| Plugin-provider export | 1 | 0 | 0 | 0 | 1 |
| **All export declarations accounted for** | **291** | **272** | **16** | **2** | **1** |

“Direct” does not require a one-to-one public Go function. A native call may
be internal plumbing behind a safer Go method, but the binding invokes that
0.17.1 declaration and preserves its behavior.

## Semantic equivalents

These 16 declarations are intentionally represented without calling that
specific C entry point:

| C declaration | Go representation |
|---|---|
| `WasmEdge_LogSetErrorLevel` | `SetLogLevel(LogLevelError)` |
| `WasmEdge_LogSetDebugLevel` | `SetLogLevel(LogLevelDebug)` |
| `WasmEdge_StringCreateByCString` | Go strings plus the length-aware internal string bridge |
| `WasmEdge_StringWrap` | Go strings plus the length-aware internal string bridge |
| `WasmEdge_StringIsEqual` | Go string equality |
| `WasmEdge_StringCopy` | conversion to an owned Go string |
| `WasmEdge_ConfigureRemoveHostRegistration` | declarative `Config`: omitted registrations are disabled |
| `WasmEdge_FunctionInstanceCreate` | `NewFunction` and the package-owned callback trampoline |
| `WasmEdge_FunctionInstanceGetData` | the opaque callback-token registry; no unsafe host-data pointer is exposed |
| `WasmEdge_MemoryInstanceGetPointerConst` | copying `Memory.Read` or explicitly unsafe `Memory.UnsafeSlice` |
| `WasmEdge_VMRunWasmFromFile` | `VM.RunFile`, implemented through native async start plus terminal `Wait` |
| `WasmEdge_VMRunWasmFromBytes` | `VM.RunBytes`, implemented through native async start plus terminal `Wait` |
| `WasmEdge_VMRunWasmFromASTModule` | `VM.Run`, implemented through native async start plus terminal `Wait` |
| `WasmEdge_LoaderParseFromBuffer` | `Loader.LoadBytes`, using the stable `WasmEdge_Bytes` entry point |
| `WasmEdge_Driver_ArgvCreate` | Go already supplies UTF-8 `[]string`; the binding builds a checked C argv |
| `WasmEdge_Driver_ArgvDelete` | the same Go-owned argv bridge frees its allocation after each driver call |

The synchronous VM calls are represented through the async C API so the
binding can provide one cancellation, lifetime, and in-flight-operation model
for synchronous, manual async, and `context.Context` calls.

## Intentional and conditional omissions

### Unsafe forced module deletion

`WasmEdge_VMForceDeleteRegisteredModule` is intentionally not exposed. The
0.17.1 header warns that it does not check module dependencies and can cause
undefined behavior or crashes. The binding instead keeps explicit dependency
leases and only permits safe teardown.

### WASI-NN RPC driver

`WasmEdge_Driver_WasiNNRPCServer` is declared only when WasmEdge itself is
built with `WASMEDGE_BUILD_WASI_NN_RPC`. The official 0.17.1 SDK artifacts do
not export it, so this package cannot link it unconditionally. Supporting it
later requires a dedicated build tag, a custom SDK/runtime built with the
same feature, and a matching CI lane.

### Plugin-provider descriptor

`WasmEdge_Plugin_GetDescriptor` is implemented *by a native plugin provider*;
it is not called by an application embedding the WasmEdge runtime. Authoring
native WasmEdge plugins is therefore a separate non-goal from the runtime
plugin discovery and instantiation API in `plugin.go`.

## Audit boundary

This accounting covers function exports in the ten stable public headers.
Enums and other value-level ABI are covered separately by the Go constants,
type tests, the exact runtime version gate, and the five corrected 0.17 Limit
ABI symbol checks in CI. A future WasmEdge C API line must repeat this audit
against its released tag before the binding widens its accepted runtime
range.

[headers]: https://github.com/WasmEdge/WasmEdge/tree/0.17.1/include/api/wasmedge

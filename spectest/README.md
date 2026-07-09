# WASM Spec Tests

This package runs the WebAssembly specification test suites against the
WasmEdge Go bindings, mirroring the behavior of the WasmEdge C API spec tests
(`test/api` and `test/spec` in the [WasmEdge](https://github.com/WasmEdge/WasmEdge)
repository).

The suites are the wast2json conversions maintained in
[WasmEdge/wasmedge-spectest](https://github.com/WasmEdge/wasmedge-spectest).
They are downloaded automatically on the first run and cached under
`testdata/`. Set `WASMEDGE_SPECTEST_PATH` to use a pre-downloaded suite.

## Running

```bash
# Interpreter mode only (fast):
go test ./spectest/ -run TestSpecInterpreter

# All three modes (interpreter, AOT, and JIT):
go test ./spectest/ -timeout 60m

# A single suite folder or unit:
go test ./spectest/ -run 'TestSpecInterpreter/wasm-1.0'
go test ./spectest/ -run 'TestSpecJIT/wasm-3.0-simd/simd_splat'
```

The AOT mode compiles every module with the native output format before
loading it, and the JIT mode enables `RunMode_JIT` on the VM configuration.
Both are skipped with `-short` and when the WasmEdge library is built without
the LLVM-based compiler.

## Known deviations from the C++ spec tests

* The `component-model` folder is skipped: these bindings do not support the
  component model.
* Fine-grained GC heap types of returned references (`eqref`, `structref`,
  `arrayref`, `i31ref`, ...) are matched as generic references with the
  correct null-ness, because the WasmEdge C API does not expose heap type
  codes or payloads of internal references.
* Host-created `anyref` argument values cannot be constructed through the C
  API; the few assertions using them are skipped and logged.
* `assert_exhaustion` commands are not checked, matching the C++ spec test
  behavior.

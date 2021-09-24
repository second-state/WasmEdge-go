### v0.9.0-rc1 (2021-09-24)

Fixed issues:

* Fixed the bugs in the load-WASM-from-buffer functions.
* Fixed the bugs in bindgen execution functions.

### v0.8.2 (2021-09-09)

Features:

* Updated to the [WasmEdge 0.8.2](https://github.com/WasmEdge/WasmEdge/releases/tag/0.8.2).
* Added the CI for build testing with every tags.

Fixed issues:

* Fixed the GC slice in host function implementation.

Docmentation:

* Added the golang version notification.
* Added the example links.

### v0.8.1 (2021-06-25)

Features:

* Updated to the [WasmEdge 0.8.1](https://github.com/WasmEdge/WasmEdge/releases/tag/0.8.1).
* WasmEdge Golang binding for C API
  * Added the new APIs about compiler options.
  * Added the new APIs about `wasmedge_process` settings.

### v0.8.0 (2021-06-01)

Features:

* WasmEdge Golang binding for C API
  * Please refer to the [README](https://github.com/second-state/WasmEdge-go/blob/master/README.md) for installation.
  * Update to the [WasmEdge 0.8.0](https://github.com/WasmEdge/WasmEdge/releases/tag/0.8.0).
* WasmEdge-go for tensorflow extension
  * The extension of [WasmEdge-tensorflow](https://github.com/second-state/WasmEdge-tensorflow) for supplying the tensorflow related host functions.
  * Please refer to the [MTCNN](https://github.com/second-state/WasmEdge-go-examples/tree/master/go_mtcnn) example.
* WasmEdge-go for image extension
  * The extension of [WasmEdge-image](https://github.com/second-state/WasmEdge-image) for supplying the image related host functions.
* Wasm-bindgen for Golang
  * Support Wasm-bindgen in WasmEdge-go.
  * Please refer to the [BindgenFuncs](https://github.com/second-state/WasmEdge-go-examples/tree/master/go_BindgenFuncs) example.

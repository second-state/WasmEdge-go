# WasmEdge for Go Package

The [WasmEdge](https://github.com/WasmEdge/WasmEdge) is a high performance WebAssembly runtime optimized for server side applications. This project provides a [golang](https://golang.org/) package for accessing to WasmEdge.

* For a complete tutorial, please read the article [Extend Your Golang App with Embedded WebAssembly Functions in WasmEdge](https://www.secondstate.io/articles/extend-golang-app-with-webassembly-rust/), which demonstrates how to embed a Wasm function and how to embed a full Wasm program from the Golang app.
* WasmEdge in real-time data strems: [AI Inference for Real-time Data Streams with WasmEdge and YoMo](https://www.secondstate.io/articles/yomo-wasmedge-real-time-data-streams/)
* For more examples, please go to the [WasmEdge-go-examples repo](https://github.com/second-state/WasmEdge-go-examples). Contributing your own example is much appreciated.

## Getting Started

The `WasmEdge-go` requires `golang` version >= `1.22`. Please check your `golang` version before installation.
Developers can [download golang here](https://golang.org/dl/).

```bash
$ go version
go version go1.23.1 linux/amd64
```

Developers must [install the WasmEdge shared library](https://wasmedge.org/docs/start/install) with the same `WasmEdge-go` release version.

```bash
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash -s -- -v 0.14.0
```

For the developers need the `WasmEdge-TensorFlow` or `WasmEdge-Image` plug-ins for `WasmEdge-go`, please install the `WasmEdge` with the corresponding plug-ins:

```bash
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash -s -- --plugins wasmedge_tensorflow wasmedge_tensorflowlite wasmedge_image -v 0.14.0
```

> Note: Please refer to the [install guide for plug-ins](https://wasmedge.org/docs/start/install/#install-wasmedge-plug-ins-and-dependencies) to check that you've installed the plug-ins with their dependencies.

For examples, please refer to the [example repository](https://github.com/second-state/WasmEdge-go-examples/).

## WasmEdge-go Documentation

Please refer to the [API Documentation](https://wasmedge.org/docs/embed/go/reference/latest) for details.

## Bazel Support on Windows

To use this library with Bazel on Windows, you can define the WasmEdge C library as a local dependency. Below is an example of how to configure this in your project.

### Example Configuration

```starlark
load("@bazel_tools//tools/build_defs/cc:cc_import.bzl", "cc_import")

cc_import(
    name = "libwasmedge",
    shared_library = "C:/wasmedge/bin/wasmedge.dll",
    interface_library = "C:/wasmedge/lib/wasmedge.lib",
    system_provided = False,
)

cc_library(
    name = "wasmedge_c",
    hdrs = [":wasmedge_headers"],
    deps = [":libwasmedge"],
    includes = ["C:/wasmedge/include"],
    visibility = ["//visibility:public"],
)

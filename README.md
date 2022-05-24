# WasmEdge for Go Package

The [WasmEdge](https://github.com/WasmEdge/WasmEdge) is a high performance WebAssembly runtime optimized for server side applications. This project provides a [golang](https://golang.org/) package for accessing to WasmEdge.

* For a complete tutorial, please read the article [Extend Your Golang App with Embedded WebAssembly Functions in WasmEdge](https://www.secondstate.io/articles/extend-golang-app-with-webassembly-rust/), which demonstrates how to embed a Wasm function and how to embed a full Wasm program from the Golang app.
* WasmEdge in real-time data strems: [AI Inference for Real-time Data Streams with WasmEdge and YoMo](https://www.secondstate.io/articles/yomo-wasmedge-real-time-data-streams/)
* For more examples, please go to the [WasmEdge-go-examples repo](https://github.com/second-state/WasmEdge-go-examples). Contributing your own example is much appreciated.

## Getting Started

The `WasmEdge-go` requires `golang` version >= `1.16`. Please check your `golang` version before installation.
Developers can [download golang here](https://golang.org/dl/).

```bash
$ go version
go version go1.16.5 linux/amd64
```

Developers must [install the WasmEdge shared library](https://wasmedge.org/book/en/start/install.html) with the same `WasmEdge-go` release version.

```bash
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash -s -- -v 0.10.0-rc.1
```

For the developers need the `TensorFlow` or `Image` extension for `WasmEdge-go`, please install the `WasmEdge` with extensions:

```bash
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash -s -- -e all -v 0.10.0-rc.1
```

Noticed that the `TensorFlow` and `Image` extensions are only for the `Linux` platforms.

**The `TensorFlow` and `Image` extensions for `darwin x86_64` platforms are in the preview stage and unstable now.**

Install the `WasmEdge-go` package and build in your Go project directory:

```bash
go get github.com/second-state/WasmEdge-go/wasmedge@v0.10.0-rc.1
go build
```

## WasmEdge-go Extensions

By default, the `WasmEdge-go` only turns on the basic runtime.

`WasmEdge-go` has the following extensions:

* Tensorflow
  * This extension supports the host functions in [WasmEdge-tensorflow](https://github.com/second-state/WasmEdge-tensorflow).
  * The `TensorFlow` extension when installing `WasmEdge` is required. Please install `WasmEdge` with the `-e tensorflow` command.
  * For using this extension, the tag `tensorflow` when building is required:

    ```bash
    go build -tags tensorflow
    ```

* Image
  * This extension supports the host functions in [WasmEdge-image](https://github.com/second-state/WasmEdge-image).
  * The `Image` extension when installing `WasmEdge` is required. Please install `WasmEdge` with the `-e image` command.
  * For using this extension, the tag `image` when building is required:

    ```bash
    go build -tags image
    ```

Users can also turn on the multiple extensions when building:

```bash
go build -tags image,tensorflow
```

For examples, please refer to the [example repository](https://github.com/second-state/WasmEdge-go-examples/).

## WasmEdge-go Documentation

Please refer to the [API Documentation](https://wasmedge.org/book/en/embed/go/ref.html) for details.

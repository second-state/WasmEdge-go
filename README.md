# WasmEdge for Go Package

The [WasmEdge](https://github.com/WasmEdge/WasmEdge) (formerly SSVM) is a high performance WebAssembly runtime optimized for server side applications. This project provides a [golang](https://golang.org/) package for accessing to WasmEdge. 

* For a complete tutorial, please read the article [Extend Your Golang App with Embedded WebAssembly Functions in WasmEdge](https://www.secondstate.io/articles/extend-golang-app-with-webassembly-rust/), which demonstrates how to embed a Wasm function and how to embed a full Wasm program from the Golang app.
* WasmEdge in real-time data strems: [AI Inference for Real-time Data Streams with WasmEdge and YoMo](https://www.secondstate.io/articles/yomo-wasmedge-real-time-data-streams/)
* For more examples, please go to the [WasmEdge-go-examples repo](https://github.com/second-state/WasmEdge-go-examples). Contributing your own example is much appreciated.

## Getting Started

The `WasmEdge-go` requires `golang` version >= `1.15`. Please check your `golang` version before installation.
Developers can [download golang here](https://golang.org/dl/).

```bash
$ go version
go version go1.16.5 linux/amd64
```

By default, the `WasmEdge-go` only turns on the basic runtime, and the [Installation of `WasmEdge` Shared Library](https://github.com/second-state/WasmEdge-go#wasmedge-shared-library-installation) is required.

Install the `WasmEdge-go` package and build in your Go project directory:

```bash
$ go get github.com/second-state/WasmEdge-go
$ go build
```

`WasmEdge-go` has the following extensions:

 - Tensorflow
    * This extension supports the host functions in [WasmEdge-tensorflow](https://github.com/second-state/WasmEdge-tensorflow).
    * For turning on this extension, the [Installation of `WasmEdge-tensorflow` Shared Library](https://github.com/second-state/WasmEdge-go#wasmedge-tensorflow-shared-library-installation) is required.
    * For using this extension, the tag `tensorflow` when building is required:
        ```bash
        $ go build -tags tensorflow
        ```
 - Image
    * This extension supports the host functions in [WasmEdge-image](https://github.com/second-state/WasmEdge-image).
    * For turning on this extension, the [Installation of `WasmEdge-image` Shared Library](https://github.com/second-state/WasmEdge-go#wasmedge-image-shared-library-installation) is required.
    * For using this extension, the tag `image` when building is required:
        ```bash
        $ go build -tags image
        ```

Users can also turn on the multiple extensions when building:

```bash
$ go build -tags image,tensorflow
```

For examples, please refer to the [example repository](https://github.com/second-state/WasmEdge-go-examples/).

## WasmEdge shared library installation

For `Ubuntu` or `Debian`:

```bash
$ wget https://github.com/WasmEdge/WasmEdge/releases/download/0.8.2/WasmEdge-0.8.2-ubuntu20.04_amd64.deb
$ sudo dpkg -i WasmEdge-0.8.2.deb
```

Or running the script:

```bash
$ wget https://github.com/second-state/WasmEdge-go/releases/download/v0.8.2/install_wasmedge.sh
$ sudo ./install_wasmedge.sh /usr/local
```

## WasmEdge-tensorflow shared library installation

```bash
$ wget https://github.com/second-state/WasmEdge-go/releases/download/v0.8.2/install_wasmedge_tensorflow_deps.sh
$ wget https://github.com/second-state/WasmEdge-go/releases/download/v0.8.2/install_wasmedge_tensorflow.sh
$ sudo ./install_wasmedge_tensorflow_deps.sh /usr/local
$ sudo ./install_wasmedge_tensorflow.sh /usr/local
```

## WasmEdge-image shared library installation

When linking with `Wasmedge-image`, `libjpeg.so.8` and `libpng16.so.16` are required.

For `ubuntu 18.04` or later, the following commands can install these dependencies:
```bash
$ sudo apt-get update
$ sudo apt-get install -y libjpeg-dev libpng-dev
```

Or you can download and install the pre-built shared libraries for the `manylinux1` platforms:

```bash
$ wget https://github.com/second-state/WasmEdge-go/releases/download/v0.8.2/install_wasmedge_image_deps.sh
$ sudo ./install_wasmedge_image_deps.sh /usr/local
```

Finally, install the `WasmEdge-image`:

```bash
$ wget https://github.com/second-state/WasmEdge-go/releases/download/v0.8.2/install_wasmedge_image.sh
$ sudo ./install_wasmedge_image.sh /usr/local
```

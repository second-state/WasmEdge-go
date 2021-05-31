# WasmEdge-Go Tensorflow Extension MTCNN example

## Build

Before building this project, please ensure the dependencies of [WasmEdge-image extension](https://github.com/second-state/WasmEdge-go#wasmedge-image-extension) and [WasmEdge-tensorflow extension](https://github.com/second-state/WasmEdge-go#wasmedge-tensorflow-extension) are installed.

```bash
# In the current directory.
$ go get -u github.com/second-state/WasmEdge-go/wasmedge
$ go build --tags image --tags tensorflow
```

## (Optional) Build the example WASM from rust

The pre-built WASM from rust is provided as "rust_mtcnn.wasm".
The pre-built and compiled WASM for AOT mode is provided as "rust_mtcnn.wasm.so".

For building the WASM from the rust source, the following steps are required:

* Install the [rust and cargo](https://www.rust-lang.org/tools/install).
* Install the `wasm32-wasi` target: `$ rustup target add wasm32-wasi`

```bash
$ cd rust_mtcnn
$ cargo build --release --target=wasm32-wasi
# The output wasm will be at `target/wasm32-wasi/release/rust_mtcnn.wasm`.
```

## Run

```bash
# For interpreter mode:
$ ./mtcnn rust_mtcnn.wasm mtcnn.pb solvay.jpg out.jpg
# For AOT mode:
$ ./mtcnn rust_mtcnn.wasm.so mtcnn.pb solvay.jpg out.jpg
```

The standard output of this example will be the following:

```bash
Go: Args: [./mtcnn rust_mtcnn.wasm mtcnn.pb solvay.jpg out.jpg]
Drawing box: 30 results ...
```

And the output image will be at `out.jpg`.

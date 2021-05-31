# WasmEdge-Go Tensorflow Extension MobileNet example

## Build

Before building this project, please ensure the dependency of [WasmEdge-tensorflow extension](https://github.com/second-state/WasmEdge-go#wasmedge-tensorflow-extension) has been installed.

```bash
# In the current directory.
$ go get -u github.com/second-state/WasmEdge-go/wasmedge
$ go build --tags tensorflow
```

## (Optional) Build the example WASM from rust

The pre-built WASM from rust is provided as "rust_mobilenet_lib_bg.wasm".
The pre-built compiled-WASM from rust is provided as "rust_mobilenet_lib_bg.so".

For building the WASM from the rust source, the following steps are required:

* Install the [rustc and cargo](https://www.rust-lang.org/tools/install).
* Set the default `rustup` version to `1.50.0` or lower.
  * `$ rustup default 1.50.0`
* Install the [rustwasmc](https://github.com/second-state/rustwasmc)
  * `$ curl https://raw.githubusercontent.com/second-state/rustwasmc/master/installer/init.sh -sSf | sh`

```bash
$ cd rust_mobilenet
$ rustwasmc build --enable-aot
# The output WASM will be `pkg/rust_mobilenet_lib_bg.wasm`.
# The output compiled-WASM will be `pkg/rust_mobilenet_lib_bg.so`.
```

## Run

```bash
# For interpreter mode:
$ ./mobilenet rust_mobilenet_lib_bg.wasm grace_hopper.jpg
# For AOT mode:
$ ./mobilenet rust_mobilenet_lib_bg.wasm.so grace_hopper.jpg
```

The standard output of this example will be the following:

```bash
Go: Args: [./mobilenet rust_mobilenet_lib_bg.wasm.so grace_hopper.jpg]
RUST: Loaded image in ... 16.522151ms
RUST: Resized image in ... 19.440301ms
RUST: Parsed output in ... 285.83336ms
RUST: index 653, prob 0.43212935
RUST: Finished post-processing in ... 285.995153ms
GO: Run bindgen -- infer: ["military uniform","medium"]
```

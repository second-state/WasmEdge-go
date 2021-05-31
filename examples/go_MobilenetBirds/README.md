# WasmEdge-Go Tensorflow Extension MobileNet-Birds example

## Build

Before building this project, please ensure the dependency of [WasmEdge-tensorflow extension](https://github.com/second-state/WasmEdge-go#wasmedge-tensorflow-extension) has been installed.

```bash
# In the current directory.
$ go get -u github.com/second-state/WasmEdge-go/wasmedge
$ go build --tags tensorflow
```

## (Optional) Build the example WASM from rust

The pre-built WASM from rust is provided as "rust_mobilenet_bird_lib_bg.wasm".
The pre-built compiled-WASM from rust is provided as "rust_mobilenet_bird_lib_bg.so".

For building the WASM from the rust source, the following steps are required:

* Install the [rustc and cargo](https://www.rust-lang.org/tools/install).
* Set the default `rustup` version to `1.50.0` or lower.
  * `$ rustup default 1.50.0`
* Install the [rustwasmc](https://github.com/second-state/rustwasmc)
  * `$ curl https://raw.githubusercontent.com/second-state/rustwasmc/master/installer/init.sh -sSf | sh`

```bash
$ cd rust_mobilenet_birds
$ rustwasmc build --enable-aot
# The output WASM will be `pkg/rust_mobilenet_bird_lib_bg.wasm`.
# The output compiled-WASM will be `pkg/rust_mobilenet_bird_lib_bg.so`.
```

## Run

```bash
# For interpreter mode:
$ ./mobilenet_birds rust_mobilenet_birds_lib_bg.wasm PurpleGallinule.jpg
# For AOT mode:
$ ./mobilenet_birds rust_mobilenet_birds_lib_bg.so PurpleGallinule.jpg
```

The standard output of this example will be the following:

```bash
Go: Args: [./mobilenet_birds rust_mobilenet_birds_lib_bg.so PurpleGallinule.jpg]
RUST: Loaded image in ... 20.342422ms
RUST: Resized image in ... 22.613784ms
RUST: Parsed output in ... 294.92347ms
RUST: index 576, prob 0.88216174
RUST: Finished post-processing in ... 295.065167ms
GO: Run bindgen -- infer: It is very likely a <a href='https://www.google.com/search?q=Porphyrio martinicus'>Porphyrio martinicus</a> in the picture
```

# WasmEdge-Go Parse image example

## Build

```bash
# In the current directory.
$ go get -u github.com/second-state/WasmEdge-go/wasmedge
$ go build
```

## (Optional) Build the example WASM from rust

The pre-built WASM from rust is provided as "rust_parseimage.wasm".

For building the WASM from the rust source, the following steps are required:

* Install the [rust and cargo](https://www.rust-lang.org/tools/install).
* Install the `wasm32-wasi` target: `$ rustup target add wasm32-wasi`

```bash
$ cd rust_ParseImage
$ cargo build --release --target=wasm32-wasi
# The output wasm will be at `target/wasm32-wasi/release/rust_parseimage.wasm`.
```

## Run

```bash
$ ./parse_image rust_parseimage.wasm < bird.jpeg
```

The standard output of this example will be the following:

```bash
Rust: Read 229104 bytes from stdin.
Rust: Got image data from stdin. Width: 1024, Height: 768
```

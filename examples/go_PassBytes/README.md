# SSVM-Go Pass bytes example

## Build

```bash
# In the current directory.
$ go get -u github.com/second-state/ssvm-go/ssvm
$ go build
```

## (Optional) Build the example WASM from rust

The pre-built WASM from rust is provided as "rust_passbytes.wasm".

For building the WASM from the rust source, the following steps are required:

* Install the [rust and cargo](https://www.rust-lang.org/tools/install).
* Install the `wasm32-wasi` target: `$ rustup target add wasm32-wasi`

```bash
$ cd rust_passbytes
$ cargo build --release --target=wasm32-wasi
# The output wasm will be at `target/wasm32-wasi/release/rust_passbytes.wasm`.
```

## Run

```bash
$ ./pass_bytes rust_passbytes.wasm bird.jpeg
```

The standard output of this example will be the following:

```bash
Go: Args: [./pass_bytes rust_passbytes.wasm bird.jpeg]
Go: Read image from file: bird.jpeg
# The temp file name is random generated and will be removed after termination.
Go: Write to temp file: /tmp/tmp.482252292.bin
Go: Start to run WASM: rust_passbytes.wasm
Rust: Process start.
Rust: Got image data from stdin. Width: 1024, Height: 768
Rust: Process end.
```

Another example is to pass the image through `stdin`:

```bash
$ ./pass_bytes rust_passbytes.wasm < bird.jpeg
Go: Args: [./pass_bytes rust_passbytes.wasm]
Go: Read image from stdin
Go: Write to temp file: /tmp/tmp.635022828.bin
Go: Start to run WASM: rust_passbytes.wasm
Rust: Process start.
Rust: Got image data from stdin. Width: 1024, Height: 768
Rust: Process end.
```

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

* Install the [rustc](https://www.rust-lang.org/tools/install).
* Install the `wasm32-wasi` target: `$ rustup target add wasm32-wasi`

```bash
$ rustc rust_passbytes.rs --target wasm32-wasi -C opt-level=3
# The output file will be `rust_passbytes.wasm`.
```

## Run

```bash
$ ./pass_bytes rust_passbytes.wasm
```

The standard output of this example will be the following:

```bash
# The temp file name is random generated and will be removed after termination.
Go: Write to temp file: /tmp/tmp.118042074.bin
Rust: Process start.
# Bytes: "Hello world!"
Rust: Got bytes [68, 65, 6C, 6C, 6F, 20, 77, 6F, 72, 6C, 64, 21]
Rust: Process end.
```

Another example is to pass through the command line arguments for querying the indices:

```bash
$ ./pass_bytes rust_passbytes.wasm 0 3 5 6 10 15
Go: Write to temp file: /tmp/tmp.320812758.bin
Rust: Process start.
Rust: Got bytes [68, 65, 6C, 6C, 6F, 20, 77, 6F, 72, 6C, 64, 21]
Rust: The index 0 of bytes is 68
Rust: The index 3 of bytes is 6C
Rust: The index 5 of bytes is 20
Rust: The index 6 of bytes is 77
Rust: The index 10 of bytes is 64
Rust: ERROR - Index "15" is out of range.
Rust: Process end.
```

# WasmEdge-Go Read file example

## Build

```bash
# In the current directory.
$ go get -u github.com/second-state/WasmEdge-go
$ go build
```

## (Optional) Build the example WASM from rust

The pre-built WASM from rust is provided as "rust_readfile.wasm".

For building the WASM from the rust source, the following steps are required:

* Install the [rustc](https://www.rust-lang.org/tools/install).
* Install the `wasm32-wasi` target: `$ rustup target add wasm32-wasi`

```bash
$ rustc rust_readfile.rs --target wasm32-wasi -C opt-level=3
# The output file will be `rust_readfile.wasm`.
```

## Run

```bash
$ ./read_file rust_readfile.wasm file.txt
```

The standard output of this example will be the following:

```bash
Rust: Opening input file "file.txt"...
Rust: Read input file "file.txt" succeeded.
Rust: Please input the line number to print the line of file.
```

Then the developers can interact with this program with standard input:

```bash
# Input "5" and press Enter.
5
# The output will be the 5th line of `file.txt`:
abcDEF___!@#$%^
# Input "-3" and press Enter.
-3
# The output will specify the error:
Rust: ERROR - Input "-3" is not an integer: invalid digit found in string
# Input "15" and press Enter.
15
# The output will specify that the file is smaller than 15 lines:
Rust: ERROR - Line "15" is out of range.
# Input "abcd" and press Enter.
abcd
# The output will specify the error:
Rust: ERROR - Input "abcd" is not an integer: invalid digit found in string
# To terminate the program, send the EOF (Ctrl + D).
^D
# The output will print the terminate message:
Rust: Process end.
```

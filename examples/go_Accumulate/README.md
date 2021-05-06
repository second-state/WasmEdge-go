# SSVM-Go Accumulate static variable example

## Build

```bash
# In the current directory.
$ go get -u github.com/second-state/ssvm-go/ssvm
$ go build
```

## (Optional) Build the example WASM from rust

The pre-built WASM from rust is provided as "accumulate_bg.wasm".

For building the WASM from the rust source, the following steps are required:

* Install the [rustc](https://www.rust-lang.org/tools/install).
* Install the `wasm32-wasi` target: `$ rustup target add wasm32-wasi`
* Install the [rustwasmc](https://github.com/second-state/rustwasmc).

```bash
$ rustwasmc build
# The output file will be `pkg/accumulate_bg.wasm`.
```

## Run

```bash
$ ./accumulate accumulate_bg.wasm
```

The standard output of this example will be the following:

```bash
1
2
3
```

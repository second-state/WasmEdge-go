# ACTIVE DEVELOMENT. Use this project with your own risk.

# Second State WebAssembly VM for Go Package

The [WasmEdge](https://github.com/WasmEdge/WasmEdge) (formerly SSVM) is a high performance WebAssembly runtime optimized for server side applications. This project provides a [golang](https://golang.org/) package for accessing to WasmEdge.

# Getting Started

## Install the WasmEdge shared library

### Option 1: Build from the source

Please refer to the [WasmEdge](https://github.com/WasmEdge/WasmEdge) for details.

```bash
# Get WasmEdge source code
$ cd <path/to/workspace/folder>
$ git clone https://github.com/WasmEdge/WasmEdge.git
$ cd WasmEdge
$ git checkout master

# Build WasmEdge source in docker
$ docker run -it --rm -v <path/to/workspace/folder>/WasmEdge:/root/WasmEdge WasmEdge/WasmEdge:latest
(docker)$ cd /root/WasmEdge
(docker)$ mkdir -p build && cd build
(docker)$ cmake -DCMAKE_BUILD_TYPE=Release -DBUILD_TOOLS=Off .. && make -j
(docker)$ exit

# Install the shared library
$ sudo cp <path/to/workspace/folder>/WasmEdge/build/include/api/wasmedge.h /usr/local/include
$ sudo cp <path/to/workspace/folder>/WasmEdge/build/lib/api/libwasmedge_c.so /usr/local/lib
$ sudo ldconfig
```

### Option 2: Install the pre-released library

```bash
$ wget https://github.com/second-state/WasmEdge-go/releases/download/v0.1.2/WasmEdge-0.8.0-rc1-manylinux2014_x86_64.tar.gz
$ sudo tar -C /usr/local -xzf WasmEdge-0.8.0-rc1-manylinux2014_x86_64.tar.gz
$ sudo ldconfig
```

## Install GO

Please refer to the [Go official site](https://golang.org/doc/install) for details.

## Install WasmEdge-go

```bash
$ go get github.com/second-state/WasmEdge-go
```

# Example

For examples, please refer to the [example folder](https://github.com/second-state/WasmEdge-go/examples).

* [Print Fibonacci](https://github.com/second-state/WasmEdge-go/tree/master/examples/go_PrintFibonacci): The simple example for using the ssvm-go package and the usage of host function and external references.
* [Read File](https://github.com/second-state/WasmEdge-go/tree/master/examples/go_ReadFile): The example for running the WASM file with WASI of read file and standard input and output.

# This is an UNFINISHED AND UNRELEASED PROJECT.

The tests are all not covered and the API may be changed rapidly.

# Second State WebAssembly VM for Go Package

The [Second State VM (SSVM)](https://github.com/second-state/ssvm) is a high performance WebAssembly runtime optimized for server side applications. This project provides a [golang](https://golang.org/) package for accessing to SSVM.

# Getting Started

## Install the SSVM shared library

### Option 1: Build from the source

Please refer to the [SSVM](https://github.com/second-state/ssvm) for details.

```bash
# Get SSVM source code
$ cd <path/to/workspace/folder>
$ git clone git@github.com:second-state/SSVM.git
$ cd SSVM
$ git checkout master

# Build SSVM source in docker
$ docker run -it --rm -v <path/to/workspace/folder>/SSVM:/root/ssvm secondstate/ssvm:latest
(docker)$ cd /root/ssvm
(docker)$ mkdir -p build && cd build
(docker)$ cmake -DCMAKE_BUILD_TYPE=Release .. && make -j
(docker)$ exit

# Install the shared library
$ sudo cp <path/to/workspace/folder>/SSVM/build/include/api/ssvm.h /usr/local/include
$ sudo cp <path/to/workspace/folder>/SSVM/build/lib/api/libssvm_c.so /usr/local/lib
$ sudo ldconfig
```

### Option 2: Install the pre-released library

```bash
$ wget https://github.com/second-state/ssvm-go/releases/download/v0.0.1/libssvm-0.7.4-rc1-manylinux2014_x86_64.tar.gz
$ sudo tar -C /usr/local -xzf libssvm-0.7.4-rc1-manylinux2014_x86_64.tar.gz
$ sudo ldconfig
```

## Install GO

Please refer to the [Go official site](https://golang.org/doc/install) for details.

## Install SSVM-Go

```bash
$ go get github.com/second-state/ssvm-go
```

# Example

For examples, please refer to the [example folder](https://github.com/second-state/ssvm-go/examples).

* [Print Fibonacci](https://github.com/second-state/ssvm-go/tree/master/examples/go_PrintFibonacci): The simple example for using the ssvm-go package and the usage of host function and external references.
* [Read File](https://github.com/second-state/ssvm-go/tree/master/examples/go_ReadFile): The example for running the WASM file with WASI of read file and standard input and output.

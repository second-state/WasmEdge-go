# This is an UNFINISHED AND UNRELEASED PROJECT.

The tests are all not covered and the API may be changed rapidly.

# Second State WebAssembly VM for Go Package

The [Second State VM (SSVM)](https://github.com/second-state/ssvm) is a high performance WebAssembly runtime optimized for server side applications. This project provides a [golang](https://golang.org/) package for accessing to SSVM.

# Getting Started

## Build and install SSVM shared library

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

## Install GO

Please refer to the [Go official site](https://golang.org/doc/install) for details.

## Install SSVM-Go

```bash
$ go get github.com/second-state/ssvm-go
```

# Example

For examples, please refer to the [example folder](https://github.com/second-state/ssvm-go/examples).

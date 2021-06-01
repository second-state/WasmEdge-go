# WasmEdge for Go Package

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
$ wget https://github.com/WasmEdge/WasmEdge/releases/download/0.8.0/WasmEdge-0.8.0-manylinux2014_x86_64.tar.gz
$ tar -xzf WasmEdge-0.8.0-manylinux2014_x86_64.tar.gz
$ sudo cp WasmEdge-0.8.0-Linux/include/wasmedge.h /usr/local/include
$ sudo cp WasmEdge-0.8.0-Linux/lib64/libwasmedge_c.so /usr/local/lib
$ sudo ldconfig
```

## Install GO

Please refer to the [Go official site](https://golang.org/doc/install) for details.

## Install WasmEdge-go

```bash
$ go get github.com/second-state/WasmEdge-go
```

# WasmEdge-image Extension

The `wasmedge_image` host functions extension for `WasmEdge-go`.
Please refer to the [WasmEdge-image](https://github.com/second-state/WasmEdge-image) for details.

## Install libjpeg and libpng

When linking with `Wasmedge-image`, `libjpeg.so.8` and `libpng16.so.16` are required.

For `ubuntu 18.04` or later, the following commands can install these libraries:
```bash
$ sudo apt-get update
$ sudo apt-get install -y libjpeg-dev libpng-dev
```

Or you can download and install the pre-built shared libraries for the `manylinux1` platforms:

```bash
$ wget https://github.com/second-state/wasmedge-image/releases/download/0.8.0/WasmEdge-image-deps-0.8.0-manylinux1_x86_64.tar.gz
$ sudo tar -C /usr/local/lib -zxvf WasmEdge-image-deps-0.8.0-manylinux1_x86_64.tar.gz
$ sudo ln -sf libjpeg.so.8.3.0 /usr/local/lib/libjpeg.so
$ sudo ln -sf libjpeg.so.8.3.0 /usr/local/lib/libjpeg.so.8
$ sudo ln -sf libpng16.so.16.37.0 /usr/local/lib/libpng.so
$ sudo ln -sf libpng16.so.16.37.0 /usr/local/lib/libpng16.so
$ sudo ln -sf libpng16.so.16.37.0 /usr/local/lib/libpng16.so.16
$ sudo ldconfig
```

## Install WasmEdge-image

```bash
# Download the wasmedge-image shared library
$ wget https://github.com/second-state/WasmEdge-image/releases/download/0.8.0/WasmEdge-image-0.8.0-manylinux2014_x86_64.tar.gz
# Install
$ sudo tar -C /usr/local/ -xzf WasmEdge-image-0.8.0-manylinux2014_x86_64.tar.gz
$ sudo ldconfig
```

## Build

For enabling this extension, you should build with the tags:

```bash
$ go build --tags image
```

# WasmEdge-tensorflow Extension

The `wasmedge_tensorflow` and `wasmedge_tensorflowlite` host functions extensions for `WasmEdge-go`.
Please refer to the [WasmEdge-tensorflow](https://github.com/second-state/WasmEdge-tensorflow) for details.

## Install the prebuilt tensorflow dependencies for the `manylinux2014` platforms

```bash
# Download the prebuilt tensorflow library
$ wget https://github.com/second-state/WasmEdge-tensorflow-deps/releases/download/0.8.0/WasmEdge-tensorflow-deps-TF-0.8.0-manylinux2014_x86_64.tar.gz
# Download the prebuilt tensorflow-lite library
$ wget https://github.com/second-state/WasmEdge-tensorflow-deps/releases/download/0.8.0/WasmEdge-tensorflow-deps-TFLite-0.8.0-manylinux2014_x86_64.tar.gz
# Install
$ sudo tar -C /usr/local/lib -xzf WasmEdge-tensorflow-deps-TF-0.8.0-manylinux2014_x86_64.tar.gz
$ sudo tar -C /usr/local/lib -xzf WasmEdge-tensorflow-deps-TFLite-0.8.0-manylinux2014_x86_64.tar.gz
# Make symbolic links
$ sudo ln -sf libtensorflow.so.2.4.0 /usr/local/lib/libtensorflow.so.2
$ sudo ln -sf libtensorflow.so.2 /usr/local/lib/libtensorflow.so
$ sudo ln -sf libtensorflow_framework.so.2.4.0 /usr/local/lib/libtensorflow_framework.so.2
$ sudo ln -sf libtensorflow_framework.so.2 /usr/local/lib/libtensorflow_framework.so
$ sudo ldconfig
```

## Install WasmEdge-tensorflow and WasmEdge-tensorflowlite

```bash
# Download the wasmedge-tensorflow shared libraries
$ wget https://github.com/second-state/WasmEdge-tensorflow/releases/download/0.8.0/WasmEdge-tensorflow-0.8.0-manylinux2014_x86_64.tar.gz
$ wget https://github.com/second-state/WasmEdge-tensorflow/releases/download/0.8.0/WasmEdge-tensorflowlite-0.8.0-manylinux2014_x86_64.tar.gz
# Install
$ sudo tar -C /usr/local/ -xzf WasmEdge-tensorflow-0.8.0-manylinux2014_x86_64.tar.gz
$ sudo tar -C /usr/local/ -xzf WasmEdge-tensorflowlite-0.8.0-manylinux2014_x86_64.tar.gz
$ sudo ldconfig
```

## Build

For enabling this extension, you should build with the tags:

```bash
$ go build --tags tensorflow
```

# Examples

For examples, please refer to the [example repository](https://github.com/second-state/WasmEdge-go-examples/).

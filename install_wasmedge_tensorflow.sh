#!/bin/sh

if [ $# -ne 1 ]
then
    echo "Args: ./install_wasmedge_tensorflow.sh INSTALL_PATH"
    exit 1
fi

if [ ! -d $1 ]
then
    mkdir -p $1
fi

wget https://github.com/second-state/WasmEdge-tensorflow/releases/download/0.8.2/WasmEdge-tensorflow-0.8.2-manylinux2014_x86_64.tar.gz
wget https://github.com/second-state/WasmEdge-tensorflow/releases/download/0.8.2/WasmEdge-tensorflowlite-0.8.2-manylinux2014_x86_64.tar.gz
tar -C $1 -xzf WasmEdge-tensorflow-0.8.2-manylinux2014_x86_64.tar.gz
tar -C $1 -xzf WasmEdge-tensorflowlite-0.8.2-manylinux2014_x86_64.tar.gz
rm -f WasmEdge-tensorflow-0.8.2-manylinux2014_x86_64.tar.gz
rm -f WasmEdge-tensorflowlite-0.8.2-manylinux2014_x86_64.tar.gz
ldconfig

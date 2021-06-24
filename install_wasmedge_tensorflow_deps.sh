#!/bin/sh

if [ $# -ne 1 ]
then
    echo "Args: ./install_wasmedge_tensorflow_deps.sh INSTALL_PATH"
    exit 1
fi

if [ ! -d "$1/lib" ]
then
    mkdir -p $1/lib
fi

wget https://github.com/second-state/WasmEdge-tensorflow-deps/releases/download/0.8.0/WasmEdge-tensorflow-deps-TF-0.8.0-manylinux2014_x86_64.tar.gz
wget https://github.com/second-state/WasmEdge-tensorflow-deps/releases/download/0.8.0/WasmEdge-tensorflow-deps-TFLite-0.8.0-manylinux2014_x86_64.tar.gz
tar -C $1/lib -zxvf WasmEdge-tensorflow-deps-TF-0.8.0-manylinux2014_x86_64.tar.gz
tar -C $1/lib -zxvf WasmEdge-tensorflow-deps-TFLite-0.8.0-manylinux2014_x86_64.tar.gz
rm -f WasmEdge-tensorflow-deps-TF-0.8.0-manylinux2014_x86_64.tar.gz
rm -f WasmEdge-tensorflow-deps-TFLite-0.8.0-manylinux2014_x86_64.tar.gz
ln -sf libtensorflow.so.2.4.0 $1/lib/libtensorflow.so.2
ln -sf libtensorflow.so.2 $1/lib/libtensorflow.so
ln -sf libtensorflow_framework.so.2.4.0 $1/lib/libtensorflow_framework.so.2
ln -sf libtensorflow_framework.so.2 $1/lib/libtensorflow_framework.so
ldconfig

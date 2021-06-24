#!/bin/sh

if [ $# -ne 1 ]
then
    echo "Args: ./install_wasmedge.sh INSTALL_PATH"
    exit 1
fi

if [ ! -d $1 ]
then
    mkdir -p $1
fi
if [ ! -d $1/include ]
then
    mkdir -p $1/include
fi
if [ ! -d $1/lib ]
then
    mkdir -p $1/lib
fi

wget https://github.com/WasmEdge/WasmEdge/releases/download/0.8.1/WasmEdge-0.8.1-manylinux2014_x86_64.tar.gz
tar -C $1 -xzf WasmEdge-0.8.1-manylinux2014_x86_64.tar.gz
rm -f WasmEdge-0.8.1-manylinux2014_x86_64.tar.gz
mv $1/WasmEdge-0.8.1-Linux/include/* $1/include
mv $1/WasmEdge-0.8.1-Linux/lib64/* $1/lib
rm -rf $1/WasmEdge-0.8.1-Linux
ldconfig

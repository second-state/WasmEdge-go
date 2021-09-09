#!/bin/sh

if [ $# -ne 1 ]
then
    echo "Args: ./install_wasmedge_image.sh INSTALL_PATH"
    exit 1
fi

if [ ! -d $1 ]
then
    mkdir -p $1
fi

wget https://github.com/second-state/WasmEdge-image/releases/download/0.8.2/WasmEdge-image-0.8.2-manylinux2014_x86_64.tar.gz
tar -C $1 -xzf WasmEdge-image-0.8.2-manylinux2014_x86_64.tar.gz
rm -f WasmEdge-image-0.8.2-manylinux2014_x86_64.tar.gz
ldconfig

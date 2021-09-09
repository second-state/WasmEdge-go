#!/bin/sh

if [ $# -ne 1 ]
then
    echo "Args: ./install_wasmedge_image_deps.sh INSTALL_PATH"
    exit 1
fi

if [ ! -d "$1/lib" ]
then
    mkdir -p $1/lib
fi

wget https://github.com/second-state/WasmEdge-image/releases/download/0.8.2/WasmEdge-image-deps-0.8.2-manylinux1_x86_64.tar.gz
tar -C $1/lib -zxvf WasmEdge-image-deps-0.8.2-manylinux1_x86_64.tar.gz
rm -f WasmEdge-image-deps-0.8.2-manylinux1_x86_64.tar.gz
ln -sf libjpeg.so.8.3.0 $1/lib/libjpeg.so
ln -sf libjpeg.so.8.3.0 $1/lib/libjpeg.so.8
ln -sf libpng16.so.16.37.0 $1/lib/libpng.so
ln -sf libpng16.so.16.37.0 $1/lib/libpng16.so
ln -sf libpng16.so.16.37.0 $1/lib/libpng16.so.16
ldconfig

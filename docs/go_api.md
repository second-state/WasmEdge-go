# WasmEdge Go SDK Documentation (For WasmEdge 0.8.2)

The WasmEdge Go SDK denotes an interface to embed the WasmEdge runtime by [WasmEdge C API](https://github.com/WasmEdge/WasmEdge/blob/0.8.2/include/api/wasmedge.h.in) into a Golang program. The followings are the guides to working with the WasmEdge-Go.
This documentation is working in progress for the fully Go SDK and the `WasmEdge 0.9.0`. For the details of the WasmEdge C API, please refer to the [full documentation](https://github.com/WasmEdge/WasmEdge/blob/0.8.2/include/api/wasmedge.h.in).

## Quick Start Guide for the WasmEdge runner

First, developers should [install WasmEdge](../README.md##Getting-Started).

The following is an example for running a WASM file.
Assume that the WASM file [`fibonacci.wasm`](https://github.com/WasmEdge/WasmEdge/blob/master/tools/wasmedge/examples/fibonacci.wasm) is copied into the current directory, and the Go project as following:

The Go module file:

```
module print_fibonacci

go 1.16

require github.com/second-state/WasmEdge-go v0.8.2 // indirect
```

The Go file:
```Go
package main

import (
    "fmt"
    "github.com/second-state/WasmEdge-go/wasmedge"
)

func main() {
    // Set not to print debug info
    wasmedge.SetLogErrorLevel()

    // Create configure
    var conf = wasmedge.NewConfigure()

    // Create store
    var store = wasmedge.NewStore()

    // Create VM by configure and external store
    var vm = wasmedge.NewVMWithConfigAndStore(conf, store)
    // Or you can call `wasmedge.NewVM()` with default config and store.

    // Instantiate wasm
    vm.LoadWasmFile("fibonacci.wasm")
    vm.Validate()
    vm.Instantiate()

    // Run to get fib[22] directly
    fmt.Println(" ### Running wasm::fib[", 22, "] ...")
    ret, err := vm.Execute("fib", uint32(22))
    if ret != nil {
        fmt.Println(" ### Get result: ", ret[0].(int32))
    } else {
        fmt.Println(" !!! Error: ", err.Error())
    }

    // Or you can call `vm.RunWasmFile()` to instantiate and invoke wasm function quickly.

    vm.Delete()
    conf.Delete()
    store.Delete()
}
```

Then you can compile and run: (the 22th fibonacci number is 28657 in 0-based index)

```bash
$ go get github.com/second-state/WasmEdge-go/wasmedge
$ go build
$ ./print_fibonacci
 ### Running wasm::fib[ 22 ] ...
 ### Get result:  28657
```

## Quick Start Guide for the WasmEdge AOT compiler

Please refer to the [WasmEdge AOT Compiler in GO](https://github.com/second-state/WasmEdge-go-examples/tree/master/go_WasmAOT) for details.

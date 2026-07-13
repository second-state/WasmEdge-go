package main

import (
	"fmt"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

func main() {
	wasmedge.SetLogOff()
	conf := wasmedge.NewConfigure(wasmedge.WASI)
	defer conf.Release()
	fmt.Println("WasmEdge version:", wasmedge.GetVersion())
}

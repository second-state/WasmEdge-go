package main

import (
	"fmt"
	"github.com/second-state/WasmEdge-go/wasmedge"
)

func main() {
	wasmedge.SetLogOff()
	conf := wasmedge.NewConfigure(wasmedge.WasmEdge_Configure_Default)
	conf.Release()
	fmt.Println("Success")
}


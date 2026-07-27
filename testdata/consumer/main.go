package main

import (
	"fmt"

	wasmedge "github.com/second-state/WasmEdge-go/v2"
)

func main() {
	vm, err := wasmedge.NewVM(nil)
	if err != nil {
		panic(err)
	}
	defer vm.Close()
	fmt.Println(wasmedge.Version())
}

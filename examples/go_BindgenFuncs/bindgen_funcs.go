package main

import (
	"fmt"
	"os"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

func main() {
	/// Expected Args[0]: program name (./bindgen_funcs)
	/// Expected Args[1]: wasm or wasm-so file (rust_bindgen_funcs_lib_bg.wasm))

	/// Set not to print debug info
	wasmedge.SetLogErrorLevel()

	/// Create configure
	var conf = wasmedge.NewConfigure(wasmedge.WASI)

	/// Create VM with configure
	var vm = wasmedge.NewVMWithConfig(conf)

	/// Init WASI
	var wasi = vm.GetImportObject(wasmedge.WASI)
	wasi.InitWasi(
		os.Args[1:],     /// The args
		os.Environ(),    /// The envs
		[]string{".:."}, /// The mapping directories
		[]string{},      /// The preopens will be empty
	)

	/// Instantiate wasm
	vm.LoadWasmFile(os.Args[1])
	vm.Validate()
	vm.Instantiate()

	/// Run bindgen functions
	var res interface{}
	var err error
	/// create_line: array, array, array -> array (inputs are JSON stringified)
	res, err = vm.ExecuteBindgen("create_line", wasmedge.Bindgen_return_array, []byte("{\"x\":1.5,\"y\":3.8}"), []byte("{\"x\":2.5,\"y\":5.8}"), []byte("A thin red line"))
	if err == nil {
		fmt.Println("Run bindgen -- create_line:", string(res.([]byte)))
	} else {
		fmt.Println("Run bindgen -- create_line FAILED")
	}
	/// say: array -> array
	res, err = vm.ExecuteBindgen("say", wasmedge.Bindgen_return_array, []byte("bindgen funcs test"))
	if err == nil {
		fmt.Println("Run bindgen -- say:", string(res.([]byte)))
	} else {
		fmt.Println("Run bindgen -- say FAILED")
	}
	/// obfusticate: array -> array
	res, err = vm.ExecuteBindgen("obfusticate", wasmedge.Bindgen_return_array, []byte("A quick brown fox jumps over the lazy dog"))
	if err == nil {
		fmt.Println("Run bindgen -- obfusticate:", string(res.([]byte)))
	} else {
		fmt.Println("Run bindgen -- obfusticate FAILED")
	}
	/// lowest_common_multiple: i32, i32 -> i32
	res, err = vm.ExecuteBindgen("lowest_common_multiple", wasmedge.Bindgen_return_i32, int32(123), int32(2))
	if err == nil {
		fmt.Println("Run bindgen -- lowest_common_multiple:", res.(int32))
	} else {
		fmt.Println("Run bindgen -- lowest_common_multiple FAILED")
	}
	/// sha3_digest: array -> array
	res, err = vm.ExecuteBindgen("sha3_digest", wasmedge.Bindgen_return_array, []byte("This is an important message"))
	if err == nil {
		fmt.Println("Run bindgen -- sha3_digest:", res.([]byte))
	} else {
		fmt.Println("Run bindgen -- sha3_digest FAILED")
	}
	/// keccak_digest: array -> array
	res, err = vm.ExecuteBindgen("keccak_digest", wasmedge.Bindgen_return_array, []byte("This is an important message"))
	if err == nil {
		fmt.Println("Run bindgen -- keccak_digest:", res.([]byte))
	} else {
		fmt.Println("Run bindgen -- keccak_digest FAILED")
	}

	vm.Delete()
	conf.Delete()
}

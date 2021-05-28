package main

import (
	"fmt"
	"os"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

func main() {
	/// Expected Args[0]: program name (./bindgen_wasi)
	/// Expected Args[1]: wasm or wasm-so file (rust_bindgen_wasi_lib_bg.wasm))

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
	/// get_random_i32: {} -> i32
	res, err = vm.ExecuteBindgen("get_random_i32", wasmedge.Bindgen_return_i32)
	if err == nil {
		fmt.Println("Run bindgen -- get_random_i32:", res.(int32))
	} else {
		fmt.Println("Run bindgen -- get_random_i32 FAILED")
	}
	/// get_random_bytes: {} -> array
	res, err = vm.ExecuteBindgen("get_random_bytes", wasmedge.Bindgen_return_array)
	if err == nil {
		fmt.Println("Run bindgen -- get_random_bytes:", res.([]byte))
	} else {
		fmt.Println("Run bindgen -- get_random_bytes FAILED")
	}
	/// echo: array -> array
	res, err = vm.ExecuteBindgen("echo", wasmedge.Bindgen_return_array, []byte("hello!!!!"))
	if err == nil {
		fmt.Println("Run bindgen -- echo:", string(res.([]byte)))
	} else {
		fmt.Println("Run bindgen -- echo FAILED")
	}
	/// print_env: {} -> {}
	res, err = vm.ExecuteBindgen("print_env", wasmedge.Bindgen_return_void)
	if err == nil {
		fmt.Println("Run bindgen -- print_env")
	} else {
		fmt.Println("Run bindgen -- print_env FAILED")
	}
	/// create_file: array, array -> {}
	res, err = vm.ExecuteBindgen("create_file", wasmedge.Bindgen_return_void, []byte("test.txt"), []byte("TEST MESSAGES----!@#@%@%$#!@#"))
	if err == nil {
		fmt.Println("Run bindgen -- create_file: test.txt")
	} else {
		fmt.Println("Run bindgen -- create_file FAILED")
	}

	/// read_file: array -> array
	res, err = vm.ExecuteBindgen("read_file", wasmedge.Bindgen_return_array, []byte("test.txt"))
	if err == nil {
		fmt.Println("Run bindgen -- read_file:", string(res.([]byte)))
	} else {
		fmt.Println("Run bindgen -- read_file FAILED")
	}

	/// del_file: array -> {}
	res, err = vm.ExecuteBindgen("del_file", wasmedge.Bindgen_return_void, []byte("test.txt"))
	if err == nil {
		fmt.Println("Run bindgen -- del_file: test.txt")
	} else {
		fmt.Println("Run bindgen -- del_file FAILED")
	}

	vm.Delete()
	conf.Delete()
}

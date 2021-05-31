package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

func main() {
	fmt.Println("Go: Args:", os.Args)
	/// Expected Args[0]: program name (./mobilenet_birds)
	/// Expected Args[1]: wasm or wasm-so file (rust_mobilenet_birds_lib_bg.wasm)
	/// Expected Args[2]: input image name (PurpleGallinule.jpg)

	/// Set not to print debug info
	wasmedge.SetLogErrorLevel()

	/// Set Tensorflow not to print debug info
	os.Setenv("TF_CPP_MIN_LOG_LEVEL", "3")
	os.Setenv("TF_CPP_MIN_VLOG_LEVEL", "3")

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

	/// Register WasmEdge-tensorflow
	var tfobj = wasmedge.NewTensorflowImportObject()
	var tfliteobj = wasmedge.NewTensorflowLiteImportObject()
	vm.RegisterImport(tfobj)
	vm.RegisterImport(tfliteobj)

	/// Instantiate wasm
	vm.LoadWasmFile(os.Args[1])
	vm.Validate()
	vm.Instantiate()

	/// Run bindgen functions
	/// infer: array -> array
	img, _ := ioutil.ReadFile(os.Args[2])
	res, err := vm.ExecuteBindgen("infer", wasmedge.Bindgen_return_array, img)
	if err == nil {
		fmt.Println("GO: Run bindgen -- infer:", string(res.([]byte)))
	} else {
		fmt.Println("GO: Run bindgen -- infer FAILED")
	}

	vm.Delete()
	conf.Delete()
}

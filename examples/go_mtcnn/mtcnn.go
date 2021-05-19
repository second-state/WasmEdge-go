package main

import (
	"fmt"
	"os"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

func main() {
	fmt.Println("Go: Args:", os.Args)
	/// Expected Args[0]: program name (./mtcnn)
	/// Expected Args[1]: wasm or wasm-so file (rust_mtcnn.wasm)
	/// Expected Args[2]: model file (mtcnn.pb)
	/// Expected Args[3]: input image name (solvay.jpg)
	/// Expected Args[4]: output image name

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

	/// Register WasmEdge-image and WasmEdge-tensorflow
	var imgobj = wasmedge.NewImageImportObject()
	var tfobj = wasmedge.NewTensorflowImportObject()
	var tfliteobj = wasmedge.NewTensorflowLiteImportObject()
	vm.RegisterImport(imgobj)
	vm.RegisterImport(tfobj)
	vm.RegisterImport(tfliteobj)

	/// Instantiate wasm
	vm.RunWasmFile(os.Args[1], "_start")

	vm.Delete()
	conf.Delete()
}

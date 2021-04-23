package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/second-state/ssvm-go/ssvm"
)

func main() {
	fmt.Println("Go: Args:", os.Args)

	/// Get the test image from argv[2] or stdin
	var data []byte
	var err error = nil
	if len(os.Args) > 2 {
		fmt.Println("Go: Read image from file:", os.Args[2])
		data, err = ioutil.ReadFile(os.Args[2])
		if err != nil {
			fmt.Println("Go: Read image from file", os.Args[2], "failed:", err)
			return
		}
	} else {
		fmt.Println("Go: Read image from stdin")
		stdin := bufio.NewReader(os.Stdin)
		data, err = ioutil.ReadAll(stdin)
		if err != nil {
			fmt.Println("Go: Read image from stdin failed:", err)
			return
		}
	}

	fmt.Println("Go: Start to run WASM:", os.Args[1])

	/// Set not to print debug info
	ssvm.SetLogErrorLevel()

	/// Create configure
	var conf = ssvm.NewConfigure(ssvm.REFERENCE_TYPES)
	conf.AddConfig(ssvm.WASI)

	/// Create VM with configure
	var vm = ssvm.NewVMWithConfig(conf)

	/// Instantiate wasm
	/// This function will also initialize WASI.
	/// The configuration of VM creation needs to add the `ssvm.WASI` flag.
	/// And the WASI initialization before calling this function will be replaced.
	vm.RunWasmFileWithDataAndWASI(
		os.Args[1],      /// WASM file path.
		"_start",        /// WASM function to execute.
		data,            /// Bytes to pass into WASM function.
		os.Args[1:],     /// List to init WASI argv.
		os.Environ(),    /// List to init WASI environment variables.
		[]string{".:."}, /// List to init WASI directory mappings.
		[]string{},      /// List to init WASI preopens.
	)

	vm.Delete()
	conf.Delete()
}

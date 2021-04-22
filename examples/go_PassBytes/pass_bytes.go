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

	/// Create a temp file
	tmpf, _ := ioutil.TempFile("", "tmp.*.bin")
	defer os.Remove(tmpf.Name())

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
	fmt.Println("Go: Write to temp file:", tmpf.Name())
	tmpf.Write(data)
	tmpf.Close()

	fmt.Println("Go: Start to run WASM:", os.Args[1])

	/// Set not to print debug info
	ssvm.SetLogErrorLevel()

	/// Create configure
	var conf = ssvm.NewConfigure(ssvm.REFERENCE_TYPES)
	conf.AddConfig(ssvm.WASI)

	/// Create VM with configure
	var vm = ssvm.NewVMWithConfig(conf)

	/// Init WASI (test)
	var wasi = vm.GetImportObject(ssvm.WASI)
	var args = []string{os.Args[1]}
	var envp = append(os.Environ(), "SSVM_DATA_TO_CALLEE="+tmpf.Name())
	var maps = []string{".:.", "/tmp:/tmp"}
	wasi.InitWasi(
		args,       /// The args
		envp,       /// The envs
		maps,       /// The mapping directories
		[]string{}, /// The preopens will be empty
	)

	/// Instantiate wasm
	vm.RunWasmFile(os.Args[1], "_start")

	vm.Delete()
	conf.Delete()
}

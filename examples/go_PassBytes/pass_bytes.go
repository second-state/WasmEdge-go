package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/second-state/ssvm-go/ssvm"
)

func main() {
	/// Set not to print debug info
	ssvm.SetLogErrorLevel()

	/// Create configure
	var conf = ssvm.NewConfigure(ssvm.REFERENCE_TYPES)
	conf.AddConfig(ssvm.WASI)

	/// Create VM with configure
	var vm = ssvm.NewVMWithConfig(conf)

	/// The test byte array
	var data = []byte("hello world!")

	/// Create a temp file
	tmpf, _ := ioutil.TempFile("", "tmp.*.bin")
	fmt.Println("Go: Write to temp file:", tmpf.Name())
	defer os.Remove(tmpf.Name())
	tmpf.Write(data)
	tmpf.Close()

	/// Init WASI (test)
	var wasi = vm.GetImportObject(ssvm.WASI)
	var args = append(os.Args[1:])
	var envp = append(os.Environ(), "TEMP_INPUT="+tmpf.Name())
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

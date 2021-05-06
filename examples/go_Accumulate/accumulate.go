package main

import (
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

    /// Init WASI (test)
    var wasi = vm.GetImportObject(ssvm.WASI)
    wasi.InitWasi(
        os.Args[1:],     /// The args
        os.Environ(),    /// The envs
        []string{}, /// The mapping directories
        []string{},      /// The preopens will be empty
    )

    vm.LoadWasmFile(os.Args[1])
    vm.Validate()
    vm.Instantiate()

    /// Instantiate wasm
    vm.Execute("acc")
    vm.Execute("acc")
    vm.Execute("acc")

    vm.Delete()
    conf.Delete()
}

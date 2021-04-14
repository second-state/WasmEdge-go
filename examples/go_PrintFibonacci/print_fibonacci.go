package main

import (
	"fmt"

	"github.com/second-state/ssvm-go/ssvm"
)

func HostPrint(data interface{}, mem *ssvm.Memory, param []interface{}) ([]interface{}, ssvm.Result) {
	// param[0]: external reference
	ref := param[0].(ssvm.ExternRef)
	value := ref.GetRef().(*int32)
	// param[1]: result of fibonacci
	fmt.Println(" [HostFunction] external value: ", *value, " , fibonacci number: ", param[1].(int32))
	return []interface{}{}, ssvm.Result_Success
}

func ListInsts(name interface{}, store *ssvm.Store) {
	if name == nil {
		fmt.Println(" --- Exported instances of the anonymous module")
		nf := store.ListFunction()
		fmt.Println("     --- Functions (", len(nf), ") : ", nf)
		nt := store.ListTable()
		fmt.Println("     --- Tables    (", len(nt), ") : ", nt)
		nm := store.ListMemory()
		fmt.Println("     --- Memories  (", len(nm), ") : ", nm)
		ng := store.ListGlobal()
		fmt.Println("     --- Globals   (", len(ng), ") : ", ng)
	} else {
		fmt.Println(" --- Exported instances of module:", name.(string))
		nf := store.ListFunctionRegistered(name.(string))
		fmt.Println("     --- Functions (", len(nf), ") : ", nf)
		nt := store.ListTableRegistered(name.(string))
		fmt.Println("     --- Tables    (", len(nt), ") : ", nt)
		nm := store.ListMemoryRegistered(name.(string))
		fmt.Println("     --- Memories  (", len(nm), ") : ", nm)
		ng := store.ListGlobalRegistered(name.(string))
		fmt.Println("     --- Globals   (", len(ng), ") : ", ng)
	}
}

func main() {
	/// Set not to print debug info
	ssvm.SetLogErrorLevel()

	/// Create configure
	var conf = ssvm.NewConfigure(ssvm.REFERENCE_TYPES)
	conf.AddConfig(ssvm.WASI)

	/// Create store
	var store = ssvm.NewStore()

	/// Create VM by configure and external store
	var vm = ssvm.NewVMWithConfigAndStore(conf, store)

	/// Init WASI (test)
	var wasi = vm.GetImportObject(ssvm.WASI)
	wasi.InitWasi([]string{"123", "arg2", "final"},
		[]string{"ENV1=VAL1", "ENV2=VALUE2"},
		[]string{".:.", "/usr/include:/usr/include"},
		[]string{"fibonacci.wasm"})

	/// Create import object
	var impobj = ssvm.NewImportObject("host", nil)

	/// Create host function
	var hostftype = ssvm.NewFunctionType(
		[]ssvm.ValType{ssvm.ValType_ExternRef, ssvm.ValType_I32},
		[]ssvm.ValType{})
	var hostprint = ssvm.NewHostFunction(hostftype, HostPrint, 0)

	/// Add host functions into import object
	impobj.AddHostFunction("print_val_and_res", hostprint)

	/// Register import module as module name "host"
	vm.RegisterImport(impobj)

	/// Register fibonacci wasm as module name "wasm"
	vm.RegisterWasmFile("wasm", "fibonacci.wasm")

	/// Instantiate wasm
	vm.LoadWasmFile("test.wasm")
	vm.Validate()
	vm.Instantiate()

	/// -----------logging-------------
	modlist := store.ListModule()
	fmt.Println("registered modules: ", modlist)
	ListInsts(nil, store)
	for _, name := range modlist {
		ListInsts(name, store)
	}
	/// -----------logging-------------

	/// Create external reference
	var value int32 = 123456
	refval := ssvm.NewExternRef(&value)

	/// Run print external value 123456 and fib[20]
	fmt.Println(" ### Running print_val_and_fib with fib[", 20, "] ...")
	var _, err = vm.Execute("print_val_and_fib", refval, uint32(20))
	if err != nil {
		fmt.Println(" !!! Error: ", err.Error())
	}

	/// Run print external value 876543210 and fib[21]
	value = 876543210
	fmt.Println(" ### Running print_val_and_fib with fib[", 21, "] ...")
	_, err = vm.Execute("print_val_and_fib", refval, uint32(21))
	if err != nil {
		fmt.Println(" !!! Error: ", err.Error())
	}

	/// Run fib[22] directly
	fmt.Println(" ### Running wasm::fib[", 22, "] ...")
	ret, err2 := vm.ExecuteRegistered("wasm", "fib", uint32(22))
	if err2 != nil {
		fmt.Println(" !!! Error: ", err.Error())
	} else if ret != nil {
		for _, val := range ret {
			fmt.Println(" Return value: ", val)
		}
	}

	refval.Release()
	vm.Delete()
	conf.Delete()
	store.Delete()
	impobj.Delete()
}

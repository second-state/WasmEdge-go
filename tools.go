package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import "unsafe"

// The Driver entry points expose the wasmedge CLI tools as library calls,
// for building custom `wasmedge`-flavored binaries on top of this package.
// Each takes the argv it would have received as a process (argv[0]
// included) and returns the process exit code.

func driverArgs(args []string) (**C.char, C.int, func()) {
	carr := make([]*C.char, len(args))
	for i, a := range args {
		carr[i] = C.CString(a)
	}
	release := func() {
		for _, p := range carr {
			C.free(unsafe.Pointer(p))
		}
	}
	if len(carr) == 0 {
		return nil, 0, release
	}
	return &carr[0], C.int(len(carr)), release
}

// DriverCompiler runs the `wasmedgec` AOT compiler CLI.
func DriverCompiler(args []string) int {
	argv, argc, free := driverArgs(args)
	defer free()
	return int(C.WasmEdge_Driver_Compiler(argc, argv))
}

// DriverTool runs the `wasmedge` runtime CLI.
func DriverTool(args []string) int {
	argv, argc, free := driverArgs(args)
	defer free()
	return int(C.WasmEdge_Driver_Tool(argc, argv))
}

// DriverUniTool runs the unified `wasmedge` CLI (runtime + compile
// subcommands).
func DriverUniTool(args []string) int {
	argv, argc, free := driverArgs(args)
	defer free()
	return int(C.WasmEdge_Driver_UniTool(argc, argv))
}

// TODO(intern-medium): A12 — bind the remaining driver helpers:
//
//	DriverWasiNNRPCServer(args []string) int
//	    -> WasmEdge_Driver_WasiNNRPCServer (pattern: DriverTool above)
//	Windows console/argv helpers
//	    -> WasmEdge_Driver_ArgvCreate/ArgvDelete/SetConsoleOutputCPtoUTF8,
//	       behind //go:build windows; document that they convert wchar_t
//	       argv from wmain into UTF-8 for the Driver* calls
//
// Verification is manual (drivers spawn full CLI runs): document the
// commands you ran in the PR description, per PLAN.md A12.

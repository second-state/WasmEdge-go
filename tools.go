package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import "unsafe"

// The Driver entry points expose the wasmedge CLI tools as library calls,
// for building custom `wasmedge`-flavored binaries on top of this package.
// Each takes the argv it would have received as a process (argv[0]
// included) and returns the process exit code.

const driverInvalidArgumentExitCode = 2

func driverArgs(args []string) (**C.char, func()) {
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
		return nil, release
	}
	return &carr[0], release
}

// DriverCompiler runs the `wasmedgec` AOT compiler CLI. It returns exit code
// 2 without entering the native driver if an argument contains an embedded
// NUL byte.
func DriverCompiler(args []string) int {
	if validateCStringSlice("driver argument", args) != nil {
		return driverInvalidArgumentExitCode
	}
	argc, err := checkedCIntCount("driver argument", uint64(len(args)))
	if err != nil {
		return driverInvalidArgumentExitCode
	}
	prepareDriverConsole()
	argv, free := driverArgs(args)
	defer free()
	return int(C.WasmEdge_Driver_Compiler(C.int(argc), argv))
}

// DriverTool runs the `wasmedge` runtime CLI. It returns exit code 2 without
// entering the native driver if an argument contains an embedded NUL byte.
func DriverTool(args []string) int {
	if validateCStringSlice("driver argument", args) != nil {
		return driverInvalidArgumentExitCode
	}
	argc, err := checkedCIntCount("driver argument", uint64(len(args)))
	if err != nil {
		return driverInvalidArgumentExitCode
	}
	prepareDriverConsole()
	argv, free := driverArgs(args)
	defer free()
	return int(C.WasmEdge_Driver_Tool(C.int(argc), argv))
}

// DriverUniTool runs the unified `wasmedge` CLI (runtime + compile
// subcommands). It returns exit code 2 without entering the native driver if
// an argument contains an embedded NUL byte.
func DriverUniTool(args []string) int {
	if validateCStringSlice("driver argument", args) != nil {
		return driverInvalidArgumentExitCode
	}
	argc, err := checkedCIntCount("driver argument", uint64(len(args)))
	if err != nil {
		return driverInvalidArgumentExitCode
	}
	prepareDriverConsole()
	argv, free := driverArgs(args)
	defer free()
	return int(C.WasmEdge_Driver_UniTool(C.int(argc), argv))
}

// TODO(platform): A12 — bind the remaining driver helpers when their build
// environments can be verified:
//
//	DriverWasiNNRPCServer(args []string) int
//	    -> WasmEdge_Driver_WasiNNRPCServer. The declaration is gated by
//	       WASMEDGE_BUILD_WASI_NN_RPC and the official 0.17.1 SDK artifacts
//	       do not export the symbol, so this needs an explicit build tag and
//	       a matching custom-runtime CI lane.
//	The Windows console helper is called automatically by Driver*. Go already
//	converts the process command line to UTF-8, so the C-only wchar_t argv
//	allocation helpers do not belong in the Go API.
//
// Verification is manual (drivers spawn full CLI runs): document the
// commands you ran in the PR description, per PLAN.md A12.

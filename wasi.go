package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"os"
	"runtime"
)

// WASIConfig declares the sandbox visible to a WASI module: command-line
// arguments, environment ("KEY=VALUE"), and preopened directories
// ("guest_path:host_path" or just a path mapped to itself).
type WASIConfig struct {
	Args     []string
	Envs     []string
	Preopens []string
	// Stdin/Stdout/Stderr redirect the module's standard streams to open
	// files. All three must be set to take effect; leaving them nil keeps
	// the process streams.
	Stdin, Stdout, Stderr *os.File
}

// NewWASIModule creates a standalone WASI module instance to register into
// an Executor/Store pipeline. VM users normally set Config.WASI instead and
// grab the built-in instance via VM.WASIModule.
func NewWASIModule(cfg WASIConfig) *Module {
	cargs, nargs, freeArgs := cStringArray(cfg.Args)
	defer freeArgs()
	cenvs, nenvs, freeEnvs := cStringArray(cfg.Envs)
	defer freeEnvs()
	cpres, npres, freePres := cStringArray(cfg.Preopens)
	defer freePres()

	var ptr *C.WasmEdge_ModuleInstanceContext
	if cfg.Stdin != nil && cfg.Stdout != nil && cfg.Stderr != nil {
		ptr = C.WasmEdge_ModuleInstanceCreateWASIWithFds(
			cargs, nargs, cenvs, nenvs, cpres, npres,
			C.int32_t(cfg.Stdin.Fd()), C.int32_t(cfg.Stdout.Fd()), C.int32_t(cfg.Stderr.Fd()))
	} else {
		ptr = C.WasmEdge_ModuleInstanceCreateWASI(
			cargs, nargs, cenvs, nenvs, cpres, npres)
	}
	runtime.KeepAlive(cfg)
	if ptr == nil {
		return nil
	}
	return ownedModule(ptr)
}

// WASIModule returns the VM's built-in WASI module instance (borrowed),
// available when the VM was created with Config.WASI. Use it to read the
// exit code or re-init the sandbox between runs.
func (vm *VM) WASIModule() (*Module, bool) {
	defer runtime.KeepAlive(vm)
	m := borrowedModule(C.WasmEdge_VMGetImportModuleContext(vm.ptr, C.WasmEdge_HostRegistration_Wasi), vm)
	return m, m != nil
}

// WASIExitCode returns the exit code a WASI program reported via
// proc_exit: 0 for a normal exit (or termination without proc_exit).
// The receiver must be a WASI module instance.
func (m *Module) WASIExitCode() uint32 {
	defer runtime.KeepAlive(m)
	return uint32(C.WasmEdge_ModuleInstanceWASIGetExitCode(m.ptr))
}

// TODO(intern-medium): A9 — bind the remaining WASI APIs on *Module:
//
//	WASINativeHandler(fd int32) (uint64, error)
//	    -> WasmEdge_ModuleInstanceWASIGetNativeHandler (non-zero return
//	       status => *Error with ErrCodeRuntimeError and a message naming
//	       the fd)
//	InitWASI(cfg WASIConfig)
//	    -> WasmEdge_ModuleInstanceInitWASI / InitWASIWithFds — re-inits the
//	       sandbox of an existing WASI module (typically VM.WASIModule
//	       between runs; mirror NewWASIModule's fd-selection logic)
//
// Pattern: NewWASIModule above (cStringArray + defer free). Extend
// wasi_test.go: run the ProcExit fixture twice with different Args and
// assert both the exit code and that InitWASI reset it.

// InitWasmEdgeProcess configures the wasmedge_process plugin's command
// allowlist (no-op unless the plugin is loaded). AllowAll grants every
// command; otherwise only allowedCmds run.
func InitWasmEdgeProcess(allowedCmds []string, allowAll bool) {
	ccmds, ncmds, freeCmds := cStringArray(allowedCmds)
	defer freeCmds()
	C.WasmEdge_ModuleInstanceInitWasmEdgeProcess(ccmds, ncmds, C.bool(allowAll))
}

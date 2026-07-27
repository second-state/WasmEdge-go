package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sync"
)

var (
	// ErrNotWASIModule reports use of a WASI-specific method on another
	// module kind.
	ErrNotWASIModule = errors.New("wasmedge: module is not wasi_snapshot_preview1")

	// ErrWASINativeHandlerNotFound reports that a guest file descriptor has no
	// host descriptor/handle mapping.
	ErrWASINativeHandlerNotFound = errors.New("wasmedge: WASI file descriptor is not mapped")

	// ErrIncompleteWASIStdio reports a configuration that supplies only some
	// of stdin, stdout, and stderr to RedirectWASIStdio. WasmEdge accepts
	// custom descriptors only as a complete set.
	ErrIncompleteWASIStdio = fmt.Errorf(
		"wasmedge: WASI stdin, stdout, and stderr must be set together: %w",
		ErrInvalidArgument)

	// ErrWASIStdioPolicyRequired reports WASI initialization without an
	// explicit choice among DiscardWASIStdio, InheritWASIStdio, and
	// RedirectWASIStdio.
	// Registering a VM's built-in WASI module alone leaves its fd table empty;
	// initialization never grants access to process streams implicitly.
	ErrWASIStdioPolicyRequired = fmt.Errorf(
		"wasmedge: WASI stdio policy must be explicit: %w",
		ErrInvalidArgument)

	// ErrWASIResourceMappingImmutable reports an attempt to change stdio or
	// preopens after a WASI module's first initialization. WasmEdge 0.17.1
	// updates arguments, environment, and exit state on re-init, but its fd
	// table only inserts missing entries and cannot replace existing mappings.
	ErrWASIResourceMappingImmutable = fmt.Errorf(
		"wasmedge: WASI stdio and preopens cannot change after initialization: %w",
		errors.ErrUnsupported)
)

type wasiStdioMode uint8

const (
	wasiStdioUnset wasiStdioMode = iota
	wasiStdioDiscard
	wasiStdioInherit
	wasiStdioRedirect
)

// WASIStdio is an explicit standard-stream capability for WASI
// initialization. Its zero value is deliberately invalid: construct a value
// with DiscardWASIStdio, InheritWASIStdio, or RedirectWASIStdio.
//
// The value is immutable and safe to copy. A redirected value refers to
// caller-owned files, which must remain open until the owning WASI module or
// VM generation is closed.
type WASIStdio struct {
	mode  wasiStdioMode
	files [3]*os.File
}

// DiscardWASIStdio gives the guest non-ambient standard streams: stdin
// immediately reaches EOF, while stdout and stderr are discarded. The
// binding owns the backing null-device descriptors and closes them after the
// native WASI module or VM generation is torn down.
func DiscardWASIStdio() WASIStdio {
	return WASIStdio{mode: wasiStdioDiscard}
}

// InheritWASIStdio explicitly grants the guest access to the embedding
// process's stdin, stdout, and stderr.
func InheritWASIStdio() WASIStdio {
	return WASIStdio{mode: wasiStdioInherit}
}

// RedirectWASIStdio grants the guest access to the supplied open files as
// stdin, stdout, and stderr. All three files must be non-nil. The caller
// retains ownership and must keep them open until the owning WASI module or
// VM generation is closed.
//
// WasmEdge 0.17.1 accepts this policy on Unix. It is rejected on Windows
// because the C API expects CRT file descriptors while os.File exposes a
// Windows handle.
func RedirectWASIStdio(stdin, stdout, stderr *os.File) WASIStdio {
	return WASIStdio{
		mode:  wasiStdioRedirect,
		files: [3]*os.File{stdin, stdout, stderr},
	}
}

// WASIConfig declares the sandbox visible to a WASI module: command-line
// arguments, environment ("KEY=VALUE"), and preopened directories
// ("guest_path:host_path" or just a path mapped to itself).
type WASIConfig struct {
	Args     []string
	Envs     []string
	Preopens []string
	// Stdio is required whenever a WASI module is initialized. Use
	// DiscardWASIStdio for non-ambient null streams, InheritWASIStdio to
	// grant the process streams, or RedirectWASIStdio to grant three
	// caller-owned files. The zero value is rejected rather than silently
	// inheriting host capabilities. WasmEdge 0.17.1 cannot replace an
	// initialized mapping, so later InitWASI calls must repeat the same
	// policy and files.
	Stdio WASIStdio
}

// wasiStdioResources keeps caller-owned *os.File wrappers reachable for as
// long as WasmEdge may use their descriptors and owns any descriptors opened
// for DiscardWASIStdio. releaseOwned handles descriptors that cannot be
// represented by os.File, notably Windows CRT descriptors.
type wasiStdioResources struct {
	mode             wasiStdioMode
	retained         [3]*os.File
	owned            []*os.File
	ownedDescriptors [3]int32
	expectedHandlers [3]uint64
	releaseOwned     func()
}

func wasiStdioResourcesForConfig(stdio WASIStdio, custom bool) wasiStdioResources {
	resources := wasiStdioResources{mode: stdio.mode}
	if !custom {
		return resources
	}
	resources.retained = stdio.files
	return resources
}

func (r *wasiStdioResources) release() {
	owned := r.owned
	releaseOwned := r.releaseOwned
	*r = wasiStdioResources{}
	for _, file := range owned {
		if file != nil {
			_ = file.Close()
		}
	}
	if releaseOwned != nil {
		releaseOwned()
	}
}

type wasiVMStateKey struct {
	vm         *C.WasmEdge_VMContext
	module     *C.WasmEdge_ModuleInstanceContext
	generation uint64
}

// wasiVMStates lets repeated VM.WASIModule calls in one VM generation share
// stdio roots. Its keys contain only C pointers, not *VM, so this registry
// does not suppress the VM's leak diagnostic. The registered generation owns
// one internal lease on the state and removes it at Reset/Close.
var wasiVMStates sync.Map // map[wasiVMStateKey]*wasiModuleState

// wasiModuleState is both an unforgeable WASI provenance marker and the
// owner of stdio reachability. It is reachable from a standalone Module or
// from the VM's registered generation.
type wasiModuleState struct {
	mu sync.Mutex

	stdio       wasiStdioResources
	preopens    []string
	initialized bool
	closed      bool

	generationRoots int
	registryKey     wasiVMStateKey
	registryBacked  bool
}

func newWASIModuleState(
	stdio wasiStdioResources,
	preopens []string,
	initialized bool,
) *wasiModuleState {
	return &wasiModuleState{
		stdio:       stdio,
		preopens:    append([]string(nil), preopens...),
		initialized: initialized,
	}
}

// applyInit serializes state validation with the native call. WasmEdge 0.17.1
// can update args/env and reset exit state repeatedly, but Environ::init uses
// unordered_map::emplace for stdio and preopens without first clearing FdMap.
// Requiring the exact original resource topology prevents the Go reachability
// state from diverging from the native fd table.
func (s *wasiModuleState) applyInit(
	preopens []string,
	requested WASIStdio,
	prepare func() (
		wasiStdioDescriptors,
		wasiStdioResources,
		error,
	),
	initNative func(wasiStdioDescriptors) error,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if s.initialized &&
		(!slices.Equal(s.preopens, preopens) ||
			s.stdio.mode != requested.mode ||
			requested.mode == wasiStdioRedirect &&
				s.stdio.retained != requested.files) {
		return ErrWASIResourceMappingImmutable
	}

	var (
		stdio wasiStdioDescriptors
		next  wasiStdioResources
		err   error
	)
	if s.initialized && requested.mode == wasiStdioDiscard {
		stdio = wasiStdioDescriptors{
			custom:           true,
			stdin:            s.stdio.ownedDescriptors[0],
			stdout:           s.stdio.ownedDescriptors[1],
			stderr:           s.stdio.ownedDescriptors[2],
			expectedHandlers: s.stdio.expectedHandlers,
		}
		next.mode = wasiStdioDiscard
	} else {
		stdio, next, err = prepare()
		if err != nil {
			return err
		}
	}

	if err := initNative(stdio); err != nil {
		next.release()
		return err
	}
	if s.initialized {
		// The existing roots describe the same native mappings and remain the
		// canonical ownership record.
		next.release()
		return nil
	}
	s.stdio = next
	s.preopens = append([]string(nil), preopens...)
	s.initialized = true
	return nil
}

func (s *wasiModuleState) close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	previous := s.stdio
	s.stdio = wasiStdioResources{}
	s.preopens = nil
	key := s.registryKey
	registered := s.registryBacked
	s.registryBacked = false
	s.mu.Unlock()

	if registered {
		wasiVMStates.CompareAndDelete(key, s)
	}
	previous.release()
}

// acquireLease lets a VM generation retain this internal state using the
// binding's existing generation-root machinery. It is not a user-visible
// native-resource lease.
func (s *wasiModuleState) acquireLease() (func(), error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrClosed
	}
	s.generationRoots++
	s.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			s.releaseGenerationRoot()
		})
	}, nil
}

func (s *wasiModuleState) releaseGenerationRoot() {
	s.mu.Lock()
	if s.generationRoots == 0 {
		s.mu.Unlock()
		return
	}
	s.generationRoots--
	if s.generationRoots != 0 || !s.registryBacked {
		s.mu.Unlock()
		return
	}
	s.closed = true
	previous := s.stdio
	s.stdio = wasiStdioResources{}
	s.preopens = nil
	key := s.registryKey
	s.registryBacked = false
	s.mu.Unlock()

	wasiVMStates.CompareAndDelete(key, s)
	previous.release()
}

func wasiStateForVM(
	vm *VM,
	ptr *C.WasmEdge_ModuleInstanceContext,
	generation uint64,
) (*wasiModuleState, error) {
	key := wasiVMStateKey{vm: vm.ptr, module: ptr, generation: generation}
	candidate := &wasiModuleState{
		registryKey:    key,
		registryBacked: true,
	}
	actual, loaded := wasiVMStates.LoadOrStore(key, candidate)
	state := actual.(*wasiModuleState)
	if err := vm.registered.retain(vm, state); err != nil {
		if !loaded {
			state.close()
		}
		return nil, err
	}
	return state, nil
}

// inheritedWASIState propagates trusted provenance only through a borrowed
// alias of the exact same native module. A user-created module with the WASI
// import name cannot acquire this state.
func inheritedWASIState(
	ptr *C.WasmEdge_ModuleInstanceContext,
	owner any,
) *wasiModuleState {
	for owner != nil {
		switch current := owner.(type) {
		case *Module:
			if current.ptr != ptr {
				return nil
			}
			if current.wasi != nil {
				return current.wasi
			}
			owner = current.life.owner
		case *generationGuard:
			vm, ok := current.owner.(*VM)
			if !ok || current.state != &vm.registered {
				return nil
			}
			current.assertAlive()
			for _, module := range vm.imports {
				if module != nil && module.ptr == ptr && module.wasi != nil {
					return module.wasi
				}
			}
			for _, module := range vm.storeState.retained {
				if module != nil && module.ptr == ptr && module.wasi != nil {
					return module.wasi
				}
			}
			wasiPtr := C.WasmEdge_VMGetImportModuleContext(
				vm.ptr,
				C.WasmEdge_HostRegistration_Wasi,
			)
			runtime.KeepAlive(vm)
			if wasiPtr != ptr {
				return nil
			}
			state, err := wasiStateForVM(vm, ptr, current.version)
			if err != nil {
				return nil
			}
			return state
		default:
			return nil
		}
	}
	return nil
}

func (m *Module) requireWASIState() (*wasiModuleState, error) {
	if m.wasi == nil {
		return nil, ErrNotWASIModule
	}
	return m.wasi, nil
}

// NewWASIModule creates a standalone WASI module instance to register into
// an Executor/Store pipeline. VM users normally set Config.WASI instead and
// grab the built-in instance via VM.WASIModule. WASIConfig.Stdio must
// explicitly discard, inherit, or redirect the standard streams. A missing
// or invalid policy matches ErrInvalidArgument; native allocation failure
// matches ErrUnavailable.
func NewWASIModule(cfg WASIConfig) (*Module, error) {
	if err := validateWASIConfigStrings(cfg); err != nil {
		return nil, err
	}
	stdio, resources, err := wasiStdioForConfig(cfg)
	if err != nil {
		return nil, err
	}
	cargs, nargs, freeArgs := cStringArray(cfg.Args)
	defer freeArgs()
	cenvs, nenvs, freeEnvs := cStringArray(cfg.Envs)
	defer freeEnvs()
	cpres, npres, freePres := cStringArray(cfg.Preopens)
	defer freePres()

	var ptr *C.WasmEdge_ModuleInstanceContext
	if stdio.custom {
		ptr = C.WasmEdge_ModuleInstanceCreateWASIWithFds(
			cargs, nargs, cenvs, nenvs, cpres, npres,
			C.int32_t(stdio.stdin), C.int32_t(stdio.stdout), C.int32_t(stdio.stderr))
	} else {
		ptr = C.WasmEdge_ModuleInstanceCreateWASI(
			cargs, nargs, cenvs, nenvs, cpres, npres)
	}
	if ptr == nil {
		resources.release()
		return nil, fmt.Errorf("create WASI module: %w", ErrUnavailable)
	}
	if stdio.custom {
		if err := verifyWASIStdioMapping(ptr, stdio.expectedHandlers); err != nil {
			C.WasmEdge_ModuleInstanceDelete(ptr)
			resources.release()
			runtime.KeepAlive(cfg)
			return nil, err
		}
	}
	runtime.KeepAlive(cfg)
	module := ownedModule(ptr)
	module.wasi = newWASIModuleState(resources, cfg.Preopens, true)
	return module, nil
}

// WASIModule returns the VM's built-in WASI module instance (borrowed),
// available when the VM was created with Config.WASI. Use it to read the
// exit code or initialize/reconfigure the sandbox between runs. The module's
// fd table is empty until the first successful InitWASI call, which requires
// an explicit standard-stream policy. Reset or Close invalidates the view.
func (vm *VM) WASIModule() (*Module, bool) {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	ptr := C.WasmEdge_VMGetImportModuleContext(
		vm.ptr,
		C.WasmEdge_HostRegistration_Wasi,
	)
	if ptr == nil {
		return nil, false
	}
	guard := vm.registeredGuard()
	module := borrowedModule(ptr, guard)
	return module, module.wasi != nil
}

// WASIExitCode returns the exit code a WASI program reported via
// proc_exit: 0 for a normal exit (or termination without proc_exit).
// ErrNotWASIModule reports a receiver without trusted WASI provenance.
func (m *Module) WASIExitCode() (uint32, error) {
	m.assertAlive()
	if _, err := m.requireWASIState(); err != nil {
		return 0, err
	}
	defer runtime.KeepAlive(m)
	return uint32(C.WasmEdge_ModuleInstanceWASIGetExitCode(m.ptr)), nil
}

// InitWASI updates arguments and environment and resets WASI execution state,
// including the prior proc_exit status. The first call on a VM's built-in
// module also establishes its preopens and required explicit stdio policy.
//
// WasmEdge 0.17.1 cannot replace entries in an initialized WASI fd table.
// Subsequent calls must therefore provide the same Preopens and exact same
// Stdio policy and files; changing them returns
// ErrWASIResourceMappingImmutable without changing any native state.
func (m *Module) InitWASI(cfg WASIConfig) error {
	m.assertAlive()
	state, err := m.requireWASIState()
	if err != nil {
		return err
	}
	if err := validateWASIConfigStrings(cfg); err != nil {
		return err
	}
	if err := validateWASIStdioPolicy(cfg.Stdio); err != nil {
		return err
	}
	// Validate caller-owned redirects before checking mapping immutability,
	// preserving ErrInvalidArgument for unsupported/closed descriptors.
	if cfg.Stdio.mode == wasiStdioRedirect {
		if _, err := platformWASIStdio(cfg.Stdio.files); err != nil {
			return err
		}
	}

	cargs, nargs, freeArgs := cStringArray(cfg.Args)
	defer freeArgs()
	cenvs, nenvs, freeEnvs := cStringArray(cfg.Envs)
	defer freeEnvs()
	cpres, npres, freePres := cStringArray(cfg.Preopens)
	defer freePres()
	defer runtime.KeepAlive(m)
	defer runtime.KeepAlive(cfg)

	return state.applyInit(
		cfg.Preopens,
		cfg.Stdio,
		func() (
			wasiStdioDescriptors,
			wasiStdioResources,
			error,
		) {
			return wasiStdioForConfig(cfg)
		},
		func(stdio wasiStdioDescriptors) error {
			if stdio.custom {
				C.WasmEdge_ModuleInstanceInitWASIWithFds(
					m.ptr, cargs, nargs, cenvs, nenvs, cpres, npres,
					C.int32_t(stdio.stdin), C.int32_t(stdio.stdout), C.int32_t(stdio.stderr))
				return verifyWASIStdioMapping(m.ptr, stdio.expectedHandlers)
			}
			C.WasmEdge_ModuleInstanceInitWASI(
				m.ptr, cargs, nargs, cenvs, nenvs, cpres, npres)
			return nil
		},
	)
}

func verifyWASIStdioMapping(
	ptr *C.WasmEdge_ModuleInstanceContext,
	expected [3]uint64,
) error {
	for fd := int32(0); fd < 3; fd++ {
		var native C.uint64_t
		status := uint32(C.WasmEdge_ModuleInstanceWASIGetNativeHandler(
			ptr, C.int32_t(fd), &native,
		))
		if status != 0 {
			return fmt.Errorf(
				"initialize WASI stdio: guest fd %d is not mapped: %w",
				fd, ErrUnavailable,
			)
		}
		if uint64(native) != expected[fd] {
			return fmt.Errorf(
				"initialize WASI stdio: guest fd %d maps to native handler %d, want %d: %w",
				fd, uint64(native), expected[fd], ErrUnavailable,
			)
		}
	}
	return nil
}

// WASINativeHandler returns the host file descriptor (Unix) or handle
// (Windows) mapped to guest fd. ErrNotWASIModule reports a receiver without
// trusted WASI provenance.
func (m *Module) WASINativeHandler(fd int32) (uint64, error) {
	m.assertAlive()
	if _, err := m.requireWASIState(); err != nil {
		return 0, err
	}
	defer runtime.KeepAlive(m)
	var native C.uint64_t
	switch status := uint32(C.WasmEdge_ModuleInstanceWASIGetNativeHandler(
		m.ptr, C.int32_t(fd), &native)); status {
	case 0:
		return uint64(native), nil
	case 2:
		return 0, ErrWASINativeHandlerNotFound
	default:
		return 0, ErrNotWASIModule
	}
}

func wasiStdioIncomplete(stdio WASIStdio) bool {
	return stdio.files[0] == nil ||
		stdio.files[1] == nil ||
		stdio.files[2] == nil
}

type wasiStdioDescriptors struct {
	custom                bool
	stdin, stdout, stderr int32
	expectedHandlers      [3]uint64
}

func wasiStdioForConfig(
	cfg WASIConfig,
) (wasiStdioDescriptors, wasiStdioResources, error) {
	if err := validateWASIStdioPolicy(cfg.Stdio); err != nil {
		return wasiStdioDescriptors{}, wasiStdioResources{}, err
	}
	switch cfg.Stdio.mode {
	case wasiStdioDiscard:
		stdio, resources, err := platformDiscardWASIStdio()
		if err != nil {
			return wasiStdioDescriptors{}, wasiStdioResources{}, err
		}
		resources.mode = wasiStdioDiscard
		return stdio, resources, nil
	case wasiStdioInherit:
		return wasiStdioDescriptors{},
			wasiStdioResourcesForConfig(cfg.Stdio, false), nil
	case wasiStdioRedirect:
		if wasiStdioIncomplete(cfg.Stdio) {
			return wasiStdioDescriptors{}, wasiStdioResources{},
				ErrIncompleteWASIStdio
		}
		stdio, err := platformWASIStdio(cfg.Stdio.files)
		if err != nil {
			return wasiStdioDescriptors{}, wasiStdioResources{}, err
		}
		return stdio, wasiStdioResourcesForConfig(cfg.Stdio, true), nil
	default:
		return wasiStdioDescriptors{}, wasiStdioResources{},
			ErrWASIStdioPolicyRequired
	}
}

func validateWASIStdioPolicy(stdio WASIStdio) error {
	switch stdio.mode {
	case wasiStdioDiscard, wasiStdioInherit:
		return nil
	case wasiStdioRedirect:
		if wasiStdioIncomplete(stdio) {
			return ErrIncompleteWASIStdio
		}
		return nil
	default:
		return ErrWASIStdioPolicyRequired
	}
}

func validateWASIConfigStrings(cfg WASIConfig) error {
	if err := validateCStringSlice("WASI argument", cfg.Args); err != nil {
		return err
	}
	if err := validateCStringSlice("WASI environment entry", cfg.Envs); err != nil {
		return err
	}
	return validateCStringSlice("WASI preopen", cfg.Preopens)
}

// InitWasmEdgeProcess configures the wasmedge_process plugin's command
// allowlist (no-op unless the plugin is loaded). AllowAll grants every
// command; otherwise only allowedCmds run. An embedded NUL byte in any command
// is rejected with ErrInvalidArgument.
func InitWasmEdgeProcess(allowedCmds []string, allowAll bool) error {
	if err := validateCStringSlice("allowed command", allowedCmds); err != nil {
		return err
	}
	ccmds, ncmds, freeCmds := cStringArray(allowedCmds)
	defer freeCmds()
	C.WasmEdge_ModuleInstanceInitWasmEdgeProcess(ccmds, ncmds, C.bool(allowAll))
	return nil
}

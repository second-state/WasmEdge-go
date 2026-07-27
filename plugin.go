package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import (
	"runtime"
	"unsafe"
)

// LoadPluginsFromDefaultPaths loads plugins from the WASMEDGE_PLUGIN_PATH
// environment variable, the user plugin directory and the install-tree
// plugin directories. Call once at startup, before creating VMs that
// should see plugin modules.
func LoadPluginsFromDefaultPaths() {
	C.WasmEdge_PluginLoadWithDefaultPaths()
}

// LoadPlugins loads plugins from a specific file or directory path. An
// embedded NUL byte is rejected with ErrInvalidArgument because the native
// API accepts a NUL-terminated path.
func LoadPlugins(path string) error {
	if err := validateCString("plugin path", path); err != nil {
		return err
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	C.WasmEdge_PluginLoadFromPath(cpath)
	return nil
}

// PluginNames lists the names of all loaded plugins.
func PluginNames() []string {
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_PluginListPluginsLength() },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_PluginListPlugins(buf, n)
		})
}

// Plugin is a loaded WasmEdge plugin (always borrowed from the global
// plugin registry; there is nothing to Close).
type Plugin struct {
	ptr *C.WasmEdge_PluginContext
}

func (p *Plugin) valid() bool {
	return p != nil && p.ptr != nil
}

// FindPlugin looks up a loaded plugin by name.
func FindPlugin(name string) (*Plugin, bool) {
	cname := newWEString(name)
	defer freeWEString(cname)
	ptr := C.WasmEdge_PluginFind(cname)
	if ptr == nil {
		return nil, false
	}
	return &Plugin{ptr: ptr}, true
}

// Name returns the plugin's name.
func (p *Plugin) Name() string {
	if !p.valid() {
		return ""
	}
	defer runtime.KeepAlive(p)
	return goString(C.WasmEdge_PluginGetPluginName(p.ptr))
}

// ModuleNames lists the modules that p can instantiate. The returned names
// are Go-owned copies and remain valid for the lifetime of the process.
func (p *Plugin) ModuleNames() []string {
	if !p.valid() {
		return nil
	}
	defer runtime.KeepAlive(p)
	return listStrings(
		func() C.uint32_t {
			return C.WasmEdge_PluginListModuleLength(p.ptr)
		},
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_PluginListModule(p.ptr, buf, n)
		})
}

// CreateModule instantiates one of the plugin's named modules. The caller
// owns the result and must Close it. A VM registration leases it until
// Reset/Close. A direct Store registration is automatically removed by
// Module.Close unless an instantiated dependant still imports it.
func (p *Plugin) CreateModule(name string) (*Module, error) {
	if !p.valid() {
		return nil, &Error{
			Category: ErrCategoryWASM,
			Code:     ErrCodeRuntimeError,
			Message:  "plugin module creation failed: invalid plugin",
		}
	}
	defer runtime.KeepAlive(p)
	cname := newWEString(name)
	defer freeWEString(cname)
	ptr := C.WasmEdge_PluginCreateModule(p.ptr, cname)
	if ptr == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "plugin module creation failed: " + name}
	}
	return ownedModule(ptr), nil
}

// InitWASINN initializes the loaded wasi_nn plugin with its model preloads.
// Call it after loading plugins and before creating a wasi_nn module.
// An empty or nil slice is passed to WasmEdge as a NULL, zero-length list.
// An embedded NUL byte in any preload is rejected with ErrInvalidArgument.
func InitWASINN(preloads []string) error {
	if err := validateCStringSlice("WASI-NN preload", preloads); err != nil {
		return err
	}
	cpreloads, npreloads, freePreloads := cStringArray(preloads)
	defer freePreloads()
	C.WasmEdge_PluginInitWASINN(cpreloads, npreloads)
	return nil
}

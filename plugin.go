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

// LoadPlugins loads plugins from a specific file or directory path.
func LoadPlugins(path string) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	C.WasmEdge_PluginLoadFromPath(cpath)
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
	defer runtime.KeepAlive(p)
	return goString(C.WasmEdge_PluginGetPluginName(p.ptr))
}

// CreateModule instantiates one of the plugin's named modules. The caller
// owns the result and must Close it (or hand it to a VM/Store as an
// import, keeping it alive meanwhile).
func (p *Plugin) CreateModule(name string) (*Module, error) {
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

// TODO(intern-easy): A5 — bind the remaining plugin APIs:
//
//	(*Plugin) ModuleNames() []string -> WasmEdge_PluginListModule /
//	    WasmEdge_PluginListModuleLength (pattern: PluginNames above,
//	    with the plugin pointer + KeepAlive like Name)
//	InitWASINN(preloads []string)    -> WasmEdge_PluginInitWASINN
//	    (pattern: cStringArray usage in wasi.go)
//
// Add plugin_test.go: PluginNames()/FindPlugin round-trip guarded by a
// t.Skip when no plugins are present on the host, plus a ModuleNames
// assertion against one loaded plugin (wasi_logging ships with most
// installs).

package wasmedge

// #include <wasmedge/wasmedge.h>
// #include <stdlib.h>
import "C"
import "unsafe"

type Plugin struct {
	_inner *C.WasmEdge_PluginContext
	_own   bool
}

func LoadPluginDefaultPaths() {
	C.WasmEdge_PluginLoadWithDefaultPaths()
}

func LoadPluginFromPath(name string) {
	cpath := C.CString(name)
	defer C.free(unsafe.Pointer(cpath))
	C.WasmEdge_PluginLoadFromPath(cpath)
}

func ListPlugins() []string {
	pluginlen := C.WasmEdge_PluginListPluginsLength()
	cnames := make([]C.WasmEdge_String, int(pluginlen))
	if int(pluginlen) > 0 {
		C.WasmEdge_PluginListPlugins(&cnames[0], pluginlen)
	}
	names := make([]string, int(pluginlen))
	for i := 0; i < int(pluginlen); i++ {
		names[i] = fromWasmEdgeString(cnames[i])
	}
	return names
}

func FindPlugin(name string) *Plugin {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_PluginFind(cname)
	if cinst == nil {
		return nil
	}
	return &Plugin{_inner: cinst, _own: false}
}

func (self *Plugin) ListModule() []string {
	modlen := C.WasmEdge_PluginListModuleLength(self._inner)
	cnames := make([]C.WasmEdge_String, int(modlen))
	if int(modlen) > 0 {
		C.WasmEdge_PluginListModule(self._inner, &cnames[0], modlen)
	}
	names := make([]string, int(modlen))
	for i := 0; i < int(modlen); i++ {
		names[i] = fromWasmEdgeString(cnames[i])
	}
	return names
}

func (self *Plugin) CreateModule(name string) *Module {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_PluginCreateModule(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Module{_inner: cinst, _own: true}
}

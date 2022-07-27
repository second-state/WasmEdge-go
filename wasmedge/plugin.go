package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

func LoadPluginDefaultPaths() {
	C.WasmEdge_Plugin_loadWithDefaultPluginPaths()
}

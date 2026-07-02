package wasmedge

// This file is the single home of the cgo build configuration. Keep #cgo
// directives out of every other file.
//
// The defaults assume a system-installed libwasmedge (>= 0.17.0). To build
// against a local WasmEdge checkout instead, override via the environment:
//
//	export WASMEDGE_DIR=$HOME/workspace/WasmEdge
//	export CGO_CFLAGS="-I$WASMEDGE_DIR/build/include/api"
//	export CGO_LDFLAGS="-L$WASMEDGE_DIR/build/lib/api -Wl,-rpath,$WASMEDGE_DIR/build/lib/api -lwasmedge"

/*
#cgo linux LDFLAGS: -lwasmedge
#cgo darwin LDFLAGS: -lwasmedge
#cgo windows LDFLAGS: -lwasmedge

#include <wasmedge/wasmedge.h>
*/
import "C"

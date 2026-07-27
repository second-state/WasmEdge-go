//go:build windows

package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

// prepareDriverConsole makes the native CLI entry points emit UTF-8 on the
// Windows console. Go has already converted os.Args to UTF-8 strings, so the
// C API's wchar_t argv allocation helpers are neither necessary nor useful
// at this boundary.
func prepareDriverConsole() {
	C.WasmEdge_Driver_SetConsoleOutputCPtoUTF8()
}

package wasmedge

/*
#include <wasmedge/wasmedge.h>

#if WASMEDGE_VERSION_MAJOR != 0 || WASMEDGE_VERSION_MINOR != 17 || \
    WASMEDGE_VERSION_PATCH < 1
#error "WasmEdge-go v2 requires WasmEdge headers >= 0.17.1 and < 0.18.0"
#endif

static const char *wasmedgego_headerVersion(void) { return WASMEDGE_VERSION; }
*/
import "C"

import "fmt"

// Version returns the full version string of the loaded WasmEdge shared
// library, for example "0.17.1".
func Version() string {
	return C.GoString(C.WasmEdge_VersionGet())
}

// VersionMajor returns the major version of the loaded library.
func VersionMajor() uint32 { return uint32(C.WasmEdge_VersionGetMajor()) }

// VersionMinor returns the minor version of the loaded library.
func VersionMinor() uint32 { return uint32(C.WasmEdge_VersionGetMinor()) }

// VersionPatch returns the patch version of the loaded library.
func VersionPatch() uint32 { return uint32(C.WasmEdge_VersionGetPatch()) }

// The binding is compiled against one C API line and must run against the
// same one: symbols and struct layouts may differ across minor versions, and
// a mismatch would otherwise surface as a crash in an arbitrary later call.
// Failing at process start with both versions in the message is the kindest
// possible failure mode, so this is a panic by design.
func init() {
	headerMajor := uint32(C.WASMEDGE_VERSION_MAJOR)
	headerMinor := uint32(C.WASMEDGE_VERSION_MINOR)
	if VersionMajor() != headerMajor || VersionMinor() != headerMinor || VersionPatch() < 1 {
		panic(fmt.Sprintf(
			"wasmedge: version mismatch: binding compiled against headers %s (%d.%d) "+
				"but the loaded libwasmedge is %s (%d.%d.%d); WasmEdge-go v2 requires "+
				"libwasmedge >= 0.17.1 and < 0.18.0; align the library and headers "+
				"(see CGO_CFLAGS/CGO_LDFLAGS in the package docs)",
			C.GoString(C.wasmedgego_headerVersion()), headerMajor, headerMinor,
			Version(), VersionMajor(), VersionMinor(), VersionPatch()))
	}
}

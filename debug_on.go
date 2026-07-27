//go:build wasmedge_debug

package wasmedge

import (
	"fmt"
	"os"
)

// reportLeak logs an owned wrapper that reached the garbage collector
// without an explicit Close. Native teardown is intentionally explicit-only:
// Go does not order cleanups across dependent C objects, so freeing here could
// turn an application leak into a process-level use-after-free.
func reportLeak(kind string) {
	fmt.Fprintf(os.Stderr, "wasmedge: leak: %s garbage-collected without Close\n", kind)
}

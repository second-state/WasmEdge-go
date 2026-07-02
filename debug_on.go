//go:build wasmedge_debug

package wasmedge

import (
	"fmt"
	"os"
)

// reportLeak logs an owned wrapper that reached the garbage collector
// without an explicit Close. The C object is still freed by the cleanup;
// this exists to make the leak visible during development.
func reportLeak(kind string) {
	fmt.Fprintf(os.Stderr, "wasmedge: leak: %s garbage-collected without Close\n", kind)
}

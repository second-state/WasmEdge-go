//go:build !wasmedge_debug

package wasmedge

// reportLeak is a no-op in normal builds. Build with -tags wasmedge_debug to
// log wrappers that were garbage-collected without Close.
func reportLeak(string) {}

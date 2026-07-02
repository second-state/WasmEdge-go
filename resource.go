package wasmedge

import (
	"runtime"
	"sync/atomic"
)

// lifetime tracks who owns the underlying C object of a wrapper.
//
// Three states, per SPEC §5.3:
//   - owned: we created it; Close frees it (idempotently).
//   - borrowed: a parent owns it; Close is a no-op and validity is bounded
//     by the parent.
//   - transferred: ownership moved into the engine (for example a *Function
//     added to a *Module); Close becomes a no-op so nothing double-frees.
//
// Wrappers embed lifetime and MUST follow two rules:
//
//  1. Every method that uses the C pointer ends with runtime.KeepAlive on
//     the wrapper (usually `defer runtime.KeepAlive(x)` as the first line).
//     Without it the GC may collect the wrapper — and run its cleanup —
//     while the C call is still executing on the raw pointer.
//  2. Constructors of owned wrappers call arm() with the free function so a
//     wrapper leaked without Close degrades to "freed at GC" instead of
//     "leaked forever" (and is reported under -tags wasmedge_debug).
type lifetime struct {
	owned   bool
	closed  atomic.Bool
	cleanup runtime.Cleanup
	// owner pins the owning wrapper of a borrowed view. Methods keep the
	// view alive across C calls (rule 1), and the view's reference to its
	// owner transitively keeps the owner's GC cleanup from freeing the C
	// object out from under the view. Nil for owned wrappers and for views
	// whose validity is guarded another way (CallContext's valid flag).
	owner any
}

// borrowed builds the lifetime of a non-owning view. Pass the wrapper that
// owns the underlying C object whenever one exists on the Go side.
func borrowed(owner any) lifetime { return lifetime{owner: owner} }

// arm registers the GC safety net on an owned wrapper. free must capture
// only the raw C pointer — never the wrapper itself, or it will never be
// collected.
func arm[T any](wrapper *T, l *lifetime, kind string, free func()) {
	l.owned = true
	l.cleanup = runtime.AddCleanup(wrapper, func(f func()) {
		reportLeak(kind)
		f()
	}, free)
}

// close runs free exactly once for owned wrappers. Safe on borrowed and
// already-closed wrappers.
func (l *lifetime) close(free func()) error {
	if !l.owned || !l.closed.CompareAndSwap(false, true) {
		return nil
	}
	l.cleanup.Stop()
	free()
	return nil
}

// transfer marks ownership as moved into the engine. It reports false when
// the wrapper was not transferable (borrowed, already closed, or already
// transferred), which callers turn into an error.
func (l *lifetime) transfer() bool {
	if !l.owned || !l.closed.CompareAndSwap(false, true) {
		return false
	}
	l.cleanup.Stop()
	return true
}

// alive reports whether the underlying C object may still be used through
// this wrapper.
func (l *lifetime) alive() bool { return !l.closed.Load() }

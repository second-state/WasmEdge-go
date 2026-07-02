package wasmedge

import "runtime/cgo"

// ExternRef pins a Go value so it can travel through WASM as an externref
// and come back intact. The pin holds the Go value alive independent of the
// garbage collector, so it is an owned resource: Close releases the pin.
//
// This replaces the v1 registry of live externrefs; the pin is a
// runtime/cgo.Handle whose opaque uintptr is what actually crosses the
// boundary.
type ExternRef struct {
	h    cgo.Handle
	life lifetime
}

// NewExternRef pins v and returns the owning reference. Wrap it into a WASM
// value with ExternRefValue.
func NewExternRef(v any) *ExternRef {
	r := &ExternRef{h: cgo.NewHandle(v)}
	h := r.h
	arm(r, &r.life, "ExternRef", func() { h.Delete() })
	return r
}

// borrowedExternRef wraps a handle that some open *ExternRef owns.
func borrowedExternRef(h uintptr) *ExternRef {
	return &ExternRef{h: cgo.Handle(h), life: borrowed()}
}

// Value returns the pinned Go value. It panics if the owning reference has
// been closed (the pin no longer exists).
func (r *ExternRef) Value() any {
	if !r.life.alive() {
		panic("wasmedge: ExternRef used after Close")
	}
	return r.h.Value()
}

// Close releases the pin. Borrowed views (from Value.ExternRef) are no-ops;
// closing twice is a no-op.
func (r *ExternRef) Close() error {
	return r.life.close(func() { r.h.Delete() })
}

func (r *ExternRef) handle() uintptr { return uintptr(r.h) }

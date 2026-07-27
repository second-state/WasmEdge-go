package wasmedge

import (
	"fmt"
	"runtime/cgo"
	"sync"
	"unsafe"
)

// externRefTokens maps package-allocated C token addresses to their internal
// Go state. The cgo.Handle is never exposed as a pointer, so an arbitrary
// native pointer whose numeric value happens to equal a handle cannot resolve.
var externRefTokens sync.Map // map[unsafe.Pointer]*externRefState

type externRefState struct {
	mu     sync.RWMutex
	handle cgo.Handle
	token  unsafe.Pointer
	closed bool
	leases int64
}

func (s *externRefState) assertAlive() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		panic("wasmedge: ExternRef used after Close")
	}
}

func (s *externRefState) value() any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		panic("wasmedge: ExternRef used after Close")
	}
	return s.handle.Value()
}

func (s *externRefState) pointer() unsafe.Pointer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		panic("wasmedge: ExternRef used after Close")
	}
	return s.token
}

func (s *externRefState) acquireLease() (func(), error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrClosed
	}
	s.leases++
	s.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			s.leases--
			s.mu.Unlock()
		})
	}, nil
}

func (s *externRefState) release() error {
	s.mu.Lock()
	s.closed = true
	if s.leases != 0 {
		n := s.leases
		s.closed = false
		s.mu.Unlock()
		return fmt.Errorf("%w: %d active dependant(s)", ErrInUse, n)
	}
	if s.token == nil {
		s.mu.Unlock()
		return nil
	}
	token, handle := s.token, s.handle
	s.token = nil
	s.handle = 0
	s.mu.Unlock()

	externRefTokens.CompareAndDelete(token, s)
	handle.Delete()
	deleteOpaqueToken(token)
	return nil
}

// ExternRef pins a Go value so it can travel through WASM as an externref
// and come back intact. The pin holds the Go value alive independent of the
// garbage collector, so it is an owned resource: Close releases the pin.
//
// The Go payload is pinned by a runtime/cgo.Handle held behind a separately
// allocated C token. Only that token address crosses the C boundary.
type ExternRef struct {
	state *externRefState
	life  lifetime
}

// NewExternRef pins v and returns the owning reference. Wrap it into a WASM
// value with ExternRefValue. A nil interface returns nil; use NullExternRef
// when a null WASM reference is intended. This allocation-only constructor
// panics if its native token cannot be allocated.
func NewExternRef(v any) *ExternRef {
	if v == nil {
		return nil
	}
	h := cgo.NewHandle(v)
	token := newOpaqueToken()
	if token == nil {
		h.Delete()
		panic("wasmedge: native ExternRef token allocation failed")
	}
	state := &externRefState{handle: h, token: token}
	externRefTokens.Store(token, state)
	r := &ExternRef{state: state}
	arm(r, &r.life, "ExternRef", func() {
		_ = state.release()
	})
	return r
}

// borrowedExternRef resolves only tokens currently registered by this
// package. The view pins no new Go handle; its validity follows owner and the
// original ExternRef state, as documented on Value.ExternRef.
func borrowedExternRef(token unsafe.Pointer, owner any) *ExternRef {
	raw, ok := externRefTokens.Load(token)
	if !ok {
		return nil
	}
	state, ok := raw.(*externRefState)
	if !ok {
		return nil
	}
	return &ExternRef{state: state, life: borrowed(state)}
}

func (r *ExternRef) assertAlive() {
	r.life.assertAlive("ExternRef")
	r.state.assertAlive()
}

func (r *ExternRef) acquireLease() (func(), error) {
	r.assertAlive()
	return r.state.acquireLease()
}

// Value returns the pinned Go value. It panics if the owning reference has
// been closed (the pin no longer exists).
func (r *ExternRef) Value() any {
	r.assertAlive()
	return r.state.value()
}

// Close releases the pin. Borrowed views (from Value.ExternRef) are no-ops;
// closing twice is a no-op.
func (r *ExternRef) Close() error {
	if !r.life.ownedAndOpen() {
		return nil
	}
	if err := r.state.release(); err != nil {
		return err
	}
	return r.life.close(func() {})
}

func (r *ExternRef) token() unsafe.Pointer { return r.state.pointer() }

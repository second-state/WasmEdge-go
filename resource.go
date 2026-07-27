package wasmedge

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	// ErrInvalidArgument reports an argument rejected before entering the
	// WasmEdge C API.
	ErrInvalidArgument = errors.New("wasmedge: invalid argument")

	// ErrUnavailable reports that an otherwise valid native-resource request
	// cannot be represented or fulfilled by the current host and WasmEdge
	// runtime.
	ErrUnavailable = errors.New("wasmedge: native resource unavailable")

	// ErrInUse reports an attempt to close or transfer a resource while a
	// dependent native operation still has a non-owning pointer to it.
	ErrInUse = errors.New("wasmedge: resource is still in use")

	// ErrUnusable reports a native executor whose shared cancellation token
	// raced completion. WasmEdge 0.17.1 provides no safe way to clear that
	// token without executing guest code, so the wrapper must be recreated.
	ErrUnusable = errors.New("wasmedge: native executor is unusable after a cancellation race")
)

type aliveGuard interface {
	assertAlive()
}

// referenceRetainer is implemented by wrappers whose lifetime is long enough
// to root Go-backed reference values stored in a native table or global.
// Returning false means the wrapper is only an ephemeral view (for example a
// CallContext-derived module), so storing the reference must be rejected.
type referenceRetainer interface {
	retainReference(any) bool
}

type referenceObject struct {
	kind uint8
	ptr  uintptr
}

const (
	referenceObjectTable uint8 = iota + 1
	referenceObjectGlobal
)

const referenceDefaultSlot = ^uint64(0)

type rootedReference struct {
	owner    any
	release  func()
	value    Value
	hasValue bool
}

func prepareReferenceRoot(owner any) (rootedReference, error) {
	return prepareReferenceRootFor(owner, nil)
}

func prepareReferenceRootFor(owner, container any) (rootedReference, error) {
	if owner == nil {
		return rootedReference{}, nil
	}
	if container != nil && ownerChainContains(owner, container) {
		return rootedReference{owner: owner}, nil
	}
	if module, ok := container.(*Module); ok && module != nil &&
		ownerChainContainsModule(owner, uintptr(unsafe.Pointer(module.ptr))) {
		return rootedReference{owner: owner}, nil
	}
	guard, ok := owner.(leaseGuard)
	if !ok {
		return rootedReference{}, ErrOwnership
	}
	release, err := guard.acquireLease()
	if err != nil {
		return rootedReference{}, err
	}
	return rootedReference{owner: owner, release: release}, nil
}

func (r *rootedReference) close() {
	if r.release != nil {
		r.release()
	}
	*r = rootedReference{}
}

// referenceRoots is shared by every Go view of one native table or global.
// Entries are replaced only after the corresponding native mutation succeeds.
type referenceRoots struct {
	mu        sync.Mutex
	container any
	slots     map[uint64]rootedReference
}

func (r *referenceRoots) prepare(owner any) (rootedReference, error) {
	r.mu.Lock()
	container := r.container
	r.mu.Unlock()
	return prepareReferenceRootFor(owner, container)
}

func (r *referenceRoots) rebind(container any) error {
	r.mu.Lock()
	previousRoots := make(map[uint64]rootedReference, len(r.slots))
	for slot, root := range r.slots {
		previousRoots[slot] = root
	}
	r.mu.Unlock()

	next := make(map[uint64]rootedReference, len(previousRoots))
	for slot, previous := range previousRoots {
		root, err := prepareReferenceRootFor(previous.owner, container)
		if err != nil {
			for _, prepared := range next {
				prepared.close()
			}
			return err
		}
		root.value = previous.value
		root.hasValue = previous.hasValue
		next[slot] = root
	}

	r.mu.Lock()
	previous := r.slots
	r.slots = next
	r.container = container
	r.mu.Unlock()
	for _, root := range previous {
		root.close()
	}
	return nil
}

func (r *referenceRoots) replace(slot uint64, next rootedReference) {
	r.mu.Lock()
	if r.slots == nil {
		r.slots = make(map[uint64]rootedReference)
	}
	previous := r.slots[slot]
	// Keep an explicit nil entry as a tombstone. Tables may have a rooted
	// default initializer; overwriting one slot with a null or native-only
	// reference must not fall back to that default owner's provenance.
	r.slots[slot] = next
	r.mu.Unlock()
	previous.close()
}

func (r *referenceRoots) owner(slot uint64, value Value) any {
	r.mu.Lock()
	defer r.mu.Unlock()
	root, ok := r.slots[slot]
	if !ok {
		root, ok = r.slots[referenceDefaultSlot]
	}
	if !ok || root.owner == nil {
		return nil
	}
	// Guest code can mutate exported tables and globals without going through
	// these wrappers. Use a recorded owner only while the native reference
	// still matches the value for which that owner was retained.
	if root.hasValue && !sameReference(value, root.value) {
		return nil
	}
	return root.owner
}

func (r *referenceRoots) close() {
	r.mu.Lock()
	slots := r.slots
	r.slots = nil
	r.mu.Unlock()
	for _, root := range slots {
		root.close()
	}
}

// persistentRoots conservatively retains references that cross into guest
// code, because WasmEdge may store them in guest tables/globals invisibly to
// the Go binding. Entries are deduplicated and released with the containing
// module generation.
type persistentRoots struct {
	mu      sync.Mutex
	entries map[any]rootedReference
}

func (r *persistentRoots) retain(owner any) error {
	finish, err := r.prepare([]any{owner})
	if err != nil {
		return err
	}
	finish(true)
	return nil
}

// prepare acquires every new reference lease without publishing any of them.
// The returned idempotent function atomically commits the whole batch when
// called with true, or rolls it all back when called with false. Native calls
// use this transaction so setup/preflight failures cannot leave a prefix of
// their arguments rooted forever.
func (r *persistentRoots) prepare(owners []any) (func(bool), error) {
	r.mu.Lock()
	seen := make(map[any]struct{}, len(owners))
	pending := make([]any, 0, len(owners))
	for _, owner := range owners {
		if owner == nil {
			continue
		}
		if _, exists := r.entries[owner]; exists {
			continue
		}
		if _, exists := seen[owner]; exists {
			continue
		}
		seen[owner] = struct{}{}
		pending = append(pending, owner)
	}
	r.mu.Unlock()

	prepared := make([]rootedReference, 0, len(pending))
	for _, owner := range pending {
		root, err := prepareReferenceRoot(owner)
		if err != nil {
			for i := range prepared {
				prepared[i].close()
			}
			return nil, err
		}
		prepared = append(prepared, root)
	}

	var once sync.Once
	return func(commit bool) {
		once.Do(func() {
			if !commit {
				for i := range prepared {
					prepared[i].close()
				}
				return
			}

			duplicates := make([]rootedReference, 0)
			r.mu.Lock()
			if r.entries == nil {
				r.entries = make(map[any]rootedReference)
			}
			for _, root := range prepared {
				if _, exists := r.entries[root.owner]; exists {
					duplicates = append(duplicates, root)
					continue
				}
				r.entries[root.owner] = root
			}
			r.mu.Unlock()
			for i := range duplicates {
				duplicates[i].close()
			}
		})
	}, nil
}

func (r *persistentRoots) close() {
	r.mu.Lock()
	entries := r.entries
	r.entries = nil
	r.mu.Unlock()
	for _, root := range entries {
		root.close()
	}
}

type referenceStateProvider interface {
	referenceState(referenceObject) (*referenceRoots, bool)
}

type invocationReferenceRetainer interface {
	prepareInvocationReferences([]Value) (func(bool), error)
}

func referenceState(owner any, object referenceObject) (*referenceRoots, bool) {
	provider, ok := owner.(referenceStateProvider)
	if !ok {
		return nil, false
	}
	return provider.referenceState(object)
}

func ownerChainContains(owner, target any) bool {
	for owner != nil {
		if owner == target {
			return true
		}
		switch current := owner.(type) {
		case *Function:
			owner = current.life.owner
		case *Memory:
			owner = current.life.owner
		case *Table:
			owner = current.life.owner
		case *Global:
			owner = current.life.owner
		case *Module:
			owner = current.life.owner
		case *Store:
			owner = current.life.owner
		case *Executor:
			owner = current.life.owner
		case *Statistics:
			owner = current.life.owner
		case *generationGuard:
			owner = current.owner
		default:
			return false
		}
	}
	return false
}

// ownerChainContainsModule reports whether owner ultimately belongs to the
// native module identified by target. Host-function entries use the native
// identity instead of retaining their containing *Module wrapper: the native
// module already owns the entry, so a Go edge back to the wrapper would keep
// an otherwise leaked module reachable forever.
func ownerChainContainsModule(owner any, target uintptr) bool {
	if target == 0 {
		return false
	}
	for owner != nil {
		switch current := owner.(type) {
		case *Function:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Memory:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Table:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Global:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Module:
			if current == nil {
				return false
			}
			if uintptr(unsafe.Pointer(current.ptr)) == target {
				return true
			}
			owner = current.life.owner
		case *Store:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Executor:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Statistics:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *generationGuard:
			if current == nil {
				return false
			}
			owner = current.owner
		default:
			return false
		}
	}
	return false
}

// ownerDependsOnModule reports whether the native object identified by owner
// is kept valid by an import lease on target. Adding the reverse lease from a
// host callback result would make both modules permanently ErrInUse.
func ownerDependsOnModule(owner any, target uintptr) bool {
	if target == 0 {
		return false
	}
	seen := make(map[*Module]struct{})
	for owner != nil {
		switch current := owner.(type) {
		case *Function:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Memory:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Table:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Global:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Module:
			if moduleDependsOnNative(current, target, seen) {
				return true
			}
			owner = current.life.owner
		case *Store:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Executor:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *Statistics:
			if current == nil {
				return false
			}
			owner = current.life.owner
		case *generationGuard:
			if current == nil {
				return false
			}
			owner = current.owner
		case *VM:
			if current == nil {
				return false
			}
			for _, imported := range current.imports {
				if moduleDependsOnNative(imported, target, seen) {
					return true
				}
			}
			for dependency := range current.dependencies {
				if moduleDependsOnNative(dependency, target, seen) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}

func moduleDependsOnNative(
	module *Module,
	target uintptr,
	seen map[*Module]struct{},
) bool {
	if module == nil {
		return false
	}
	if uintptr(unsafe.Pointer(module.ptr)) == target {
		return true
	}
	if _, exists := seen[module]; exists {
		return false
	}
	seen[module] = struct{}{}
	for _, dependency := range module.dependencyModules {
		if moduleDependsOnNative(dependency, target, seen) {
			return true
		}
	}
	return false
}

func callScopedOwner(owner any) bool {
	for owner != nil {
		switch current := owner.(type) {
		case *CallContext:
			return true
		case *Function:
			owner = current.life.owner
		case *Memory:
			owner = current.life.owner
		case *Table:
			owner = current.life.owner
		case *Global:
			owner = current.life.owner
		case *Module:
			owner = current.life.owner
		case *Store:
			owner = current.life.owner
		case *Executor:
			owner = current.life.owner
		case *Statistics:
			owner = current.life.owner
		case *generationGuard:
			owner = current.owner
		default:
			return false
		}
	}
	return false
}

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
//     This keeps the wrapper's Go owner graph and handles reachable through
//     the C call and prevents premature leak diagnostics.
//  2. Constructors of owned wrappers call arm() so an omitted Close is
//     reported under -tags wasmedge_debug. The cleanup deliberately does not
//     call into C: runtime cleanup order across a graph of wrappers is
//     unspecified, so nondeterministic native teardown can cause use-after-
//     free. Native resources are released only by explicit Close.
type lifetimeState struct {
	owned   bool
	closed  atomic.Bool
	leases  atomic.Int64
	cleanup runtime.Cleanup
}

// lifetime deliberately keeps native ownership state behind a pointer.
// Exported wrappers are normally used through pointers, but Go still permits
// callers to copy their values. Every such copy must observe the same
// close/transfer/lease state so the native context is deleted at most once.
type lifetime struct {
	state *lifetimeState
	// owner pins the owning wrapper of a borrowed view. Methods keep the
	// view alive across C calls (rule 1), and the view's reference to its
	// owner keeps the ownership graph reachable and enables recursive
	// validity checks. Nil for owned wrappers and for views whose validity
	// is guarded another way (CallContext's valid flag).
	owner any
}

// borrowed builds the lifetime of a non-owning view. Pass the wrapper that
// owns the underlying C object whenever one exists on the Go side.
func borrowed(owner any) lifetime { return lifetime{owner: owner} }

func (l *lifetime) isOwned() bool {
	return l.state != nil && l.state.owned
}

func (l *lifetime) isClosed() bool {
	return l.state != nil && l.state.closed.Load()
}

// assertAlive rejects use after Close/transfer and recursively checks the
// owner of a borrowed view. This makes a chain such as
// Function -> Module -> CallContext expire as one unit.
func (l *lifetime) assertAlive(kind string) {
	if l.isClosed() {
		panic(fmt.Sprintf("wasmedge: %s used after Close or ownership transfer", kind))
	}
	if owner, ok := l.owner.(aliveGuard); ok {
		owner.assertAlive()
	}
}

// arm registers leak diagnostics on an owned wrapper. free remains a
// parameter so constructors spell the explicit native destructor next to the
// allocation, but is intentionally not retained by the cleanup; see rule 2.
func arm[T any](wrapper *T, l *lifetime, kind string, free func()) {
	state := &lifetimeState{owned: true}
	l.state = state
	_ = free
	// Attach diagnostics to the shared state, not the first wrapper value.
	// A copied wrapper keeps state reachable, so collecting the original
	// wrapper cannot report a resource that is still usable through its copy.
	state.cleanup = runtime.AddCleanup(state, reportLeak, kind)
	runtime.KeepAlive(wrapper)
}

func (s *lifetimeState) stopCleanup() {
	s.cleanup.Stop()
	// Stop only guarantees removal while the object passed to AddCleanup is
	// reachable across the call.
	runtime.KeepAlive(s)
}

// close runs free exactly once for owned wrappers. Safe on borrowed and
// already-closed wrappers. A leased resource must outlive its dependants, so
// callers close those first and retry.
func (l *lifetime) close(free func()) error {
	state := l.state
	if state == nil || !state.owned || state.closed.Load() {
		return nil
	}
	if n := state.leases.Load(); n != 0 {
		return fmt.Errorf("%w: %d active dependant(s)", ErrInUse, n)
	}
	if !state.closed.CompareAndSwap(false, true) {
		return nil
	}
	state.stopCleanup()
	free()
	return nil
}

// transferable reports whether ownership may move into the engine without
// changing the lifetime. Callers that must prepare other state before the
// transfer can use this to reject borrowed, closed, or leased wrappers first.
func (l *lifetime) transferable() error {
	state := l.state
	if state == nil || !state.owned {
		return ErrOwnership
	}
	if state.closed.Load() {
		return ErrClosed
	}
	if n := state.leases.Load(); n != 0 {
		return fmt.Errorf("%w: %d active dependant(s)", ErrInUse, n)
	}
	return nil
}

// transfer marks ownership as moved into the engine. It fails before any C
// call when the wrapper is borrowed, already closed, or already transferred.
func (l *lifetime) transfer() error {
	if err := l.transferable(); err != nil {
		return err
	}
	state := l.state
	if !state.closed.CompareAndSwap(false, true) {
		return ErrClosed
	}
	state.stopCleanup()
	return nil
}

// acquire pins an owned resource against explicit Close, or delegates through
// a borrowed view to the wrapper that owns the native object. The returned
// release function is idempotent.
func (l *lifetime) acquire(kind string) (func(), error) {
	if owner, ok := l.owner.(leaseGuard); ok {
		return owner.acquireLease()
	}
	state := l.state
	if state == nil || !state.owned {
		if state != nil && state.closed.Load() {
			return nil, ErrClosed
		}
		return nil, fmt.Errorf("wasmedge: cannot lease ephemeral %s: %w", kind, ErrOwnership)
	}
	if state.closed.Load() {
		return nil, ErrClosed
	}
	state.leases.Add(1)
	if state.closed.Load() {
		state.leases.Add(-1)
		return nil, ErrClosed
	}
	var released atomic.Bool
	return func() {
		if released.CompareAndSwap(false, true) {
			state.leases.Add(-1)
		}
	}, nil
}

func (l *lifetime) ensureUnused() error {
	state := l.state
	if state != nil && state.closed.Load() {
		return ErrClosed
	}
	if state != nil {
		if n := state.leases.Load(); n != 0 {
			return fmt.Errorf("%w: %d active dependant(s)", ErrInUse, n)
		}
	}
	return nil
}

func (l *lifetime) ownedAndOpen() bool {
	state := l.state
	return state != nil && state.owned && !state.closed.Load()
}

// alive reports whether the underlying C object may still be used through
// this wrapper.
func (l *lifetime) alive() bool { return !l.isClosed() }

// leaseGuard is implemented by native wrappers that can be retained by an
// asynchronous operation or another native resource.
type leaseGuard interface {
	acquireLease() (func(), error)
}

type executionGate struct {
	active   atomic.Bool
	poisoned atomic.Bool
}

func (g *executionGate) begin() (func(), error) {
	if g.poisoned.Load() {
		return nil, ErrUnusable
	}
	if !g.active.CompareAndSwap(false, true) {
		return nil, ErrInUse
	}
	if g.poisoned.Load() {
		g.active.Store(false)
		return nil, ErrUnusable
	}
	var released atomic.Bool
	return func() {
		if released.CompareAndSwap(false, true) {
			g.active.Store(false)
		}
	}, nil
}

func (g *executionGate) poison() { g.poisoned.Store(true) }

// generation tracks native children that can be replaced while their owning
// wrapper remains open (notably a VM's active and registered modules).
type generation struct {
	value           atomic.Uint64
	roots           persistentRoots
	referenceMu     sync.Mutex
	referenceStates map[referenceObject]*referenceRoots
	retiredStates   []*referenceRoots
}

func (g *generation) invalidate() {
	g.roots.close()
	g.referenceMu.Lock()
	states := g.referenceStates
	g.referenceStates = nil
	retired := g.retiredStates
	g.retiredStates = nil
	g.referenceMu.Unlock()
	for _, state := range states {
		state.close()
	}
	for _, state := range retired {
		state.close()
	}
	g.value.Add(1)
}

// advance invalidates borrowed views after native replacement while retaining
// old reference roots until the containing VM is reset or closed. Retaining is
// conservative but necessary for one-shot runs whose failure stage does not
// reveal whether the old native module was replaced.
func (g *generation) advance() {
	g.referenceMu.Lock()
	for _, state := range g.referenceStates {
		g.retiredStates = append(g.retiredStates, state)
	}
	g.referenceStates = nil
	g.referenceMu.Unlock()
	g.value.Add(1)
}

func (g *generation) retain(container, owner any) error {
	if container != nil && ownerChainContains(owner, container) {
		return nil
	}
	return g.roots.retain(owner)
}

func (g *generation) prepareInvocationReferences(
	container any,
	owners []any,
) (func(bool), error) {
	filtered := make([]any, 0, len(owners))
	for _, owner := range owners {
		if container != nil && ownerChainContains(owner, container) {
			continue
		}
		filtered = append(filtered, owner)
	}
	return g.roots.prepare(filtered)
}

func (g *generation) state(container any, object referenceObject) *referenceRoots {
	g.referenceMu.Lock()
	defer g.referenceMu.Unlock()
	if g.referenceStates == nil {
		g.referenceStates = make(map[referenceObject]*referenceRoots)
	}
	state := g.referenceStates[object]
	if state == nil {
		state = &referenceRoots{container: container}
		g.referenceStates[object] = state
	}
	return state
}

// generationGuard bounds a borrowed native pointer by both its owner's
// lifetime and the generation in which the pointer was obtained.
type generationGuard struct {
	owner   any
	state   *generation
	version uint64
}

func guardGeneration(owner any, state *generation) *generationGuard {
	return &generationGuard{owner: owner, state: state, version: state.value.Load()}
}

func (g *generationGuard) assertAlive() {
	if owner, ok := g.owner.(aliveGuard); ok {
		owner.assertAlive()
	}
	if g.state.value.Load() != g.version {
		panic("wasmedge: borrowed native view used after its owner replaced the object")
	}
}

func (g *generationGuard) acquireLease() (func(), error) {
	g.assertAlive()
	owner, ok := g.owner.(leaseGuard)
	if !ok {
		return nil, ErrOwnership
	}
	release, err := owner.acquireLease()
	if err != nil {
		return nil, err
	}
	if g.state.value.Load() != g.version {
		release()
		return nil, ErrClosed
	}
	return release, nil
}

func (g *generationGuard) referenceState(object referenceObject) (*referenceRoots, bool) {
	g.assertAlive()
	return g.state.state(g.owner, object), true
}

func (g *generationGuard) retainReference(root any) bool {
	g.assertAlive()
	return g.state.retain(g.owner, root) == nil
}

func (g *generationGuard) prepareInvocationReferences(
	values []Value,
) (func(bool), error) {
	g.assertAlive()
	return g.state.prepareInvocationReferences(g.owner, valueOwners(values))
}

// acquireLease acquires every lease in order and rolls back on failure.
// Non-resource values are retained by the returned closure but need no
// explicit lease.
func acquireLeases(owners ...any) (func(), error) {
	releases := make([]func(), 0, len(owners))
	retained := append([]any(nil), owners...)
	for _, owner := range owners {
		if owner == nil {
			continue
		}
		if guard, ok := owner.(leaseGuard); ok {
			release, err := guard.acquireLease()
			if err != nil {
				for i := len(releases) - 1; i >= 0; i-- {
					releases[i]()
				}
				return nil, err
			}
			releases = append(releases, release)
		}
	}
	var released atomic.Bool
	return func() {
		if !released.CompareAndSwap(false, true) {
			return
		}
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
		runtime.KeepAlive(retained)
	}, nil
}

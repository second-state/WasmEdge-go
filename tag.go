package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Tag is an exception-handling tag instance (always borrowed; the C API
// offers no way to create one).
type Tag struct {
	ptr  *C.WasmEdge_TagInstanceContext
	life lifetime
}

func borrowedTag(ptr *C.WasmEdge_TagInstanceContext, owner any) *Tag {
	if ptr == nil {
		return nil
	}
	return &Tag{ptr: ptr, life: borrowed(owner)}
}

func (t *Tag) assertAlive() { t.life.assertAlive("Tag") }

// Type returns a copied descriptor of the tag's type.
func (t *Tag) Type() TagType {
	t.assertAlive()
	defer runtime.KeepAlive(t)
	tt, _ := tagTypeFromC(C.WasmEdge_TagInstanceGetTagType(t.ptr))
	return tt
}

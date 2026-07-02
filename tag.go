package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Tag is an exception-handling tag instance (always borrowed; the C API
// offers no way to create one).
type Tag struct {
	ptr   *C.WasmEdge_TagInstanceContext
	owner any // pins the module the tag was found in
}

func borrowedTag(ptr *C.WasmEdge_TagInstanceContext, owner any) *Tag {
	if ptr == nil {
		return nil
	}
	return &Tag{ptr: ptr, owner: owner}
}

// Type returns the tag's type (borrowed).
func (t *Tag) Type() *TagType {
	defer runtime.KeepAlive(t)
	return borrowedTagType(C.WasmEdge_TagInstanceGetTagType(t.ptr), t)
}

// +build image

package wasmedge

/*
#cgo linux LDFLAGS: -lwasmedge-image_c
#cgo darwin LDFLAGS: -lwasmedge-image_c

#include <wasmedge/wasmedge-image.h>
*/
import "C"

func NewImageModule() *Module {
	obj := C.WasmEdge_Image_ModuleInstanceCreate()
	if obj == nil {
		return nil
	}
	return &Module{_inner: obj, _own: true}
}

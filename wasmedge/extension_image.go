// +build image

package wasmedge

/*
#cgo linux LDFLAGS: -lwasmedge-image_c

#include <wasmedge/wasmedge-image.h>
*/
import "C"

func NewImageImportObject() *ImportObject {
	obj := C.WasmEdge_Image_ImportObjectCreate()
	if obj == nil {
		return nil
	}
	return &ImportObject{_inner: obj, _own: true}
}

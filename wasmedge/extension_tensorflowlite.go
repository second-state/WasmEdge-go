// +build tensorflowlite

package wasmedge

/*
#cgo linux LDFLAGS: -lwasmedge-tensorflowlite_c -ltensorflowlite_c
#cgo darwin LDFLAGS: -lwasmedge-tensorflowlite_c -ltensorflowlite_c

#include <wasmedge/wasmedge-tensorflowlite.h>
*/
import "C"

func NewTensorflowModule() *Module {
	obj := C.WasmEdge_Tensorflow_ModuleInstanceCreateDummy()
	if obj == nil {
		return nil
	}
	return &Module{_inner: obj, _own: true}
}

func NewTensorflowLiteModule() *Module {
	obj := C.WasmEdge_TensorflowLite_ModuleInstanceCreate()
	if obj == nil {
		return nil
	}
	return &Module{_inner: obj, _own: true}
}

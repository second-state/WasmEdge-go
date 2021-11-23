// +build tensorflow

package wasmedge

/*
#cgo linux LDFLAGS: -lwasmedge-tensorflow_c -lwasmedge-tensorflowlite_c -ltensorflow -ltensorflow_framework -ltensorflowlite_c

#include <wasmedge/wasmedge-tensorflow.h>
#include <wasmedge/wasmedge-tensorflowlite.h>
*/
import "C"

func NewTensorflowImportObject() *ImportObject {
	obj := C.WasmEdge_Tensorflow_ImportObjectCreate()
	if obj == nil {
		return nil
	}
	return &ImportObject{_inner: obj, _own: true}
}

func NewTensorflowLiteImportObject() *ImportObject {
	obj := C.WasmEdge_TensorflowLite_ImportObjectCreate()
	if obj == nil {
		return nil
	}
	return &ImportObject{_inner: obj, _own: true}
}

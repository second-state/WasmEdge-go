// +build tensorflow

package wasmedge

/*
#cgo linux LDFLAGS: -lwasmedge-tensorflow_c -lwasmedge-tensorflowlite_c -ltensorflow -ltensorflow_framework -ltensorflowlite_c

#include <wasmedge/wasmedge-tensorflow.h>
#include <wasmedge/wasmedge-tensorflowlite.h>
*/
import "C"

func NewTensorflowImportObject() *ImportObject {
	self := &ImportObject{
		_inner: C.WasmEdge_Tensorflow_ImportObjectCreate(),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewTensorflowLiteImportObject() *ImportObject {
	self := &ImportObject{
		_inner: C.WasmEdge_TensorflowLite_ImportObjectCreate(),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

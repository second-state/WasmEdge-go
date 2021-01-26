package ssvm

// #include <ssvm.h>
import "C"

type Address struct {
	_inner C.SSVM_InstanceAddress
}

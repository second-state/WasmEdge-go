package ssvm

// #include <ssvm.h>
import "C"

func GetVersion() string {
	return C.GoString(C.SSVM_VersionGet())
}

func GetVersionMajor() uint {
	return uint(C.SSVM_VersionGetMajor())
}

func GetVersionMinor() uint {
	return uint(C.SSVM_VersionGetMinor())
}

func GetVersionPatch() uint {
	return uint(C.SSVM_VersionGetPatch())
}

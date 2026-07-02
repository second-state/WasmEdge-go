package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Validator checks a loaded ASTModule against the configured WASM standard
// and proposals.
type Validator struct {
	ptr  *C.WasmEdge_ValidatorContext
	life lifetime
}

// NewValidator creates a validator honoring cfg (nil for defaults).
func NewValidator(cfg *Config) (*Validator, error) {
	ccfg, free := cfg.build()
	defer free()
	ptr := C.WasmEdge_ValidatorCreate(ccfg)
	if ptr == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError, Message: "validator creation failed"}
	}
	v := &Validator{ptr: ptr}
	arm(v, &v.life, "Validator", func() { C.WasmEdge_ValidatorDelete(ptr) })
	return v, nil
}

// Validate checks the module. A nil return marks the module as validated
// and instantiable.
func (v *Validator) Validate(m *ASTModule) error {
	defer runtime.KeepAlive(v)
	defer runtime.KeepAlive(m)
	return newResult(C.WasmEdge_ValidatorValidate(v.ptr, m.ptr))
}

// Close frees the validator. No-op after the first call.
func (v *Validator) Close() error {
	ptr := v.ptr
	return v.life.close(func() { C.WasmEdge_ValidatorDelete(ptr) })
}

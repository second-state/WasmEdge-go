package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"fmt"
	"runtime"
)

// Validator checks a loaded ASTModule against the configured WASM standard
// and proposals.
type Validator struct {
	ptr  *C.WasmEdge_ValidatorContext
	life lifetime
}

// NewValidator creates a validator honoring cfg (nil for defaults).
func NewValidator(cfg *Config) (*Validator, error) {
	ccfg, free, err := cfg.build()
	if err != nil {
		return nil, fmt.Errorf("create validator configuration: %w", err)
	}
	defer free()
	ptr := C.WasmEdge_ValidatorCreate(ccfg)
	if ptr == nil {
		return nil, fmt.Errorf("create validator: %w", ErrUnavailable)
	}
	v := &Validator{ptr: ptr}
	arm(v, &v.life, "Validator", func() { C.WasmEdge_ValidatorDelete(ptr) })
	return v, nil
}

func borrowedValidator(ptr *C.WasmEdge_ValidatorContext, owner any) *Validator {
	if ptr == nil {
		return nil
	}
	return &Validator{ptr: ptr, life: borrowed(owner)}
}

func (v *Validator) assertAlive() { v.life.assertAlive("Validator") }

// Validate checks the module. A nil return marks the module as validated
// and instantiable.
func (v *Validator) Validate(m *ASTModule) error {
	v.assertAlive()
	if m == nil {
		return fmt.Errorf("validate requires a non-nil AST module: %w", ErrInvalidArgument)
	}
	m.assertAlive()
	defer runtime.KeepAlive(v)
	defer runtime.KeepAlive(m)
	return newResult(C.WasmEdge_ValidatorValidate(v.ptr, m.ptr))
}

// Close frees the validator. No-op after the first call.
func (v *Validator) Close() error {
	ptr := v.ptr
	return v.life.close(func() { C.WasmEdge_ValidatorDelete(ptr) })
}

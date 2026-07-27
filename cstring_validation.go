package wasmedge

import (
	"fmt"
	"strings"
)

// validateCString rejects strings that cannot be represented by a NUL-
// terminated C string without changing their contents. Length-delimited
// WasmEdge_String values do not need this check.
func validateCString(label, value string) error {
	if strings.IndexByte(value, 0) < 0 {
		return nil
	}
	return fmt.Errorf("%s contains an embedded NUL byte: %w", label, ErrInvalidArgument)
}

func validateCStringSlice(label string, values []string) error {
	for i, value := range values {
		if err := validateCString(fmt.Sprintf("%s[%d]", label, i), value); err != nil {
			return err
		}
	}
	return nil
}

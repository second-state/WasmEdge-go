package wasmedge

import (
	"fmt"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	version := GetVersion()
	if version == "" {
		t.Fatal("GetVersion() returned an empty string")
	}
	if GetVersionMajor() != 0 || GetVersionMinor() != 17 {
		t.Errorf("unexpected version %d.%d, the bindings target WasmEdge 0.17.x",
			GetVersionMajor(), GetVersionMinor())
	}
	prefix := fmt.Sprintf("%d.%d.%d", GetVersionMajor(), GetVersionMinor(), GetVersionPatch())
	if !strings.HasPrefix(version, prefix) {
		t.Errorf("GetVersion() = %q does not start with %q", version, prefix)
	}
}

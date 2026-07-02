package wasmedge

import (
	"fmt"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Fatal("empty version string")
	}
	prefix := fmt.Sprintf("%d.%d.%d", VersionMajor(), VersionMinor(), VersionPatch())
	if !strings.HasPrefix(v, prefix) {
		t.Fatalf("version string %q does not start with %q", v, prefix)
	}
	// The init version gate already panicked if this binding was compiled
	// against a different C API line, so reaching this point is the test.
}

func TestLogControls(t *testing.T) {
	SetLogLevel(LogLevelError)
	SetLogCallback(func(LogMessage) {})
	SetLogCallback(nil)
	SetLogOff()
	// Callback delivery is asserted end-to-end in pipeline_test.go once the
	// Loader exists (a parse failure emits an engine log record).
}

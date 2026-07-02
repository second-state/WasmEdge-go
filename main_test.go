package wasmedge

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Several tests exercise engine error paths on purpose; silence the
	// default engine logger so expected failures do not pollute output.
	// Tests asserting log delivery install their own callback and level.
	SetLogOff()
	os.Exit(m.Run())
}

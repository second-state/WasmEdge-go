package wasmedge

import "testing"

func TestCLIVersion(t *testing.T) {
	if ret := RunWasmEdgeCLI([]string{"wasmedge", "--version"}); ret != 0 {
		t.Errorf("wasmedge --version returned %d", ret)
	}
	if ret := RunWasmEdgeAOTCompilerCLI([]string{"wasmedgec", "--version"}); ret != 0 {
		t.Errorf("wasmedgec --version returned %d", ret)
	}
	if ret := RunWasmEdgeUnifiedCLI([]string{"wasmedge", "--version"}); ret != 0 {
		t.Errorf("unified wasmedge --version returned %d", ret)
	}
}

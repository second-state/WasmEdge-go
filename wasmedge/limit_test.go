package wasmedge

import "testing"

const (
	minVal = 1024
)

func TestNewLimit(t *testing.T) {
	l := NewLimit(minVal)
	if l.min != minVal {
		t.Fatal("wrong min value")
	}
	if l.hasmax {
		t.Fatal("should have no max value")
	}

	l.WithMaxVal(minVal * 2)
	if !l.hasmax {
		t.Fatal("should have max value")
	}
}

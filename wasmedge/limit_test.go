package wasmedge

import "testing"

const (
	minVal = 1024
)

func TestNewLimit(t *testing.T) {
	l := NewLimit(minVal)
	if l.GetMin() != minVal {
		t.Fatal("wrong min value")
	}
	if l.HasMax() {
		t.Fatal("should have no max value")
	}

	l = NewLimitWithMax(minVal, minVal*2)
	if !l.HasMax() {
		t.Fatal("should have max value")
	}
}

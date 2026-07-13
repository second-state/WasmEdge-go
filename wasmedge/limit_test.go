package wasmedge

import "testing"

const (
	minVal = 1024
)

func TestNewLimit(t *testing.T) {
	l := NewLimit(minVal)
	if l == nil {
		t.Fatal("NewLimit() returned nil")
	}
	defer l.Release()
	if l.GetMin() != minVal {
		t.Fatal("wrong min value")
	}
	if l.HasMax() {
		t.Fatal("should have no max value")
	}
	if l.IsShared() {
		t.Fatal("should not be shared")
	}
	if l.Is64Bit() {
		t.Fatal("should not be 64-bit")
	}

	lmax := NewLimitWithMax(minVal, minVal*2)
	if lmax == nil {
		t.Fatal("NewLimitWithMax() returned nil")
	}
	defer lmax.Release()
	if !lmax.HasMax() {
		t.Fatal("should have max value")
	}
	if lmax.GetMax() != minVal*2 {
		t.Fatal("wrong max value")
	}

	if NewLimitWithMax(minVal, minVal-1) != nil {
		t.Fatal("max < min should be rejected")
	}
}

func TestNewLimitShared(t *testing.T) {
	l := NewLimitSharedWithMax(minVal, minVal*2)
	if l == nil {
		t.Fatal("NewLimitSharedWithMax() returned nil")
	}
	defer l.Release()
	if !l.IsShared() || !l.HasMax() {
		t.Fatal("shared limit with max expected")
	}
	if NewLimitSharedWithMax(minVal, minVal-1) != nil {
		t.Fatal("max < min should be rejected")
	}
}

func TestNewLimit64(t *testing.T) {
	l := NewLimit64(minVal)
	if l == nil {
		t.Fatal("NewLimit64() returned nil")
	}
	defer l.Release()
	if !l.Is64Bit() {
		t.Fatal("should be 64-bit")
	}
	if l.HasMax() {
		t.Fatal("should have no max value")
	}

	lmax := NewLimit64WithMax(minVal, minVal*4)
	if lmax == nil {
		t.Fatal("NewLimit64WithMax() returned nil")
	}
	defer lmax.Release()
	if !lmax.Is64Bit() || !lmax.HasMax() || lmax.GetMax() != minVal*4 {
		t.Fatal("wrong 64-bit limit with max")
	}

	lshared := NewLimit64SharedWithMax(minVal, minVal*4)
	if lshared == nil {
		t.Fatal("NewLimit64SharedWithMax() returned nil")
	}
	defer lshared.Release()
	if !lshared.Is64Bit() || !lshared.IsShared() || !lshared.HasMax() {
		t.Fatal("wrong 64-bit shared limit")
	}

	if NewLimit64WithMax(minVal, minVal-1) != nil ||
		NewLimit64SharedWithMax(minVal, minVal-1) != nil {
		t.Fatal("max < min should be rejected")
	}
}

func TestLimitIsEqual(t *testing.T) {
	l1 := NewLimitWithMax(minVal, minVal*2)
	defer l1.Release()
	l2 := NewLimitWithMax(minVal, minVal*2)
	defer l2.Release()
	l3 := NewLimitWithMax(minVal, minVal*3)
	defer l3.Release()
	l4 := NewLimit64WithMax(minVal, minVal*2)
	defer l4.Release()
	l5 := NewLimitSharedWithMax(minVal, minVal*2)
	defer l5.Release()

	if !l1.IsEqual(l2) {
		t.Error("limits with the same values should be equal")
	}
	if l1.IsEqual(l3) {
		t.Error("limits with different max values should not be equal")
	}
	if l1.IsEqual(l4) {
		t.Error("32-bit and 64-bit limits should not be equal")
	}
	if l1.IsEqual(l5) {
		t.Error("shared and unshared limits should not be equal")
	}
}

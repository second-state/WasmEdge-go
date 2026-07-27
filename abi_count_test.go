package wasmedge

import (
	"errors"
	"math"
	"testing"
)

func TestCheckedABICounts(t *testing.T) {
	if got, err := checkedUint32Count("values", math.MaxUint32); err != nil ||
		got != math.MaxUint32 {
		t.Fatalf("uint32 boundary: got %d, err=%v", got, err)
	}
	if _, err := checkedUint32Count(
		"values", uint64(math.MaxUint32)+1,
	); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("uint32 overflow: got %v, want ErrInvalidArgument", err)
	}

	if got, err := checkedCIntCount("arguments", math.MaxInt32); err != nil ||
		got != math.MaxInt32 {
		t.Fatalf("C int boundary: got %d, err=%v", got, err)
	}
	if _, err := checkedCIntCount(
		"arguments", uint64(math.MaxInt32)+1,
	); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("C int overflow: got %v, want ErrInvalidArgument", err)
	}
}

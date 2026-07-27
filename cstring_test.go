package wasmedge

import (
	"math"
	"testing"
)

func TestMustFitWEStringRejectsUnrepresentableLength(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustFitWEString did not panic above the native uint32 limit")
		}
	}()
	mustFitWEString(uint64(math.MaxUint32) + 1)
}

func TestSnapshotListRetriesAfterGrowth(t *testing.T) {
	fills := 0
	got := snapshotList(
		func() uint32 { return 1 },
		func(buf []int) uint32 {
			fills++
			if fills == 1 {
				buf[0] = 10
				return 2
			}
			buf[0], buf[1] = 10, 20
			return 2
		},
	)
	if fills != 2 {
		t.Fatalf("fill calls: got %d, want 2", fills)
	}
	if len(got) != 2 || got[0] != 10 || got[1] != 20 {
		t.Fatalf("snapshot: got %v, want [10 20]", got)
	}
}

func TestSnapshotListUsesReportedCount(t *testing.T) {
	got := snapshotList(
		func() uint32 { return 3 },
		func(buf []string) uint32 {
			buf[0] = "only"
			return 1
		},
	)
	if len(got) != 1 || got[0] != "only" {
		t.Fatalf("snapshot: got %v, want [only]", got)
	}
}

func TestSnapshotParallelListsRetriesAfterGrowth(t *testing.T) {
	fills := 0
	left, right := snapshotParallelLists(
		func() uint32 { return 1 },
		func(left []string, right []int) uint32 {
			fills++
			if fills == 1 {
				return 2
			}
			left[0], left[1] = "a", "b"
			right[0], right[1] = 10, 20
			return 2
		},
	)
	if fills != 2 {
		t.Fatalf("fill calls: got %d, want 2", fills)
	}
	if len(left) != 2 || left[0] != "a" || left[1] != "b" ||
		len(right) != 2 || right[0] != 10 || right[1] != 20 {
		t.Fatalf("snapshot: got %v/%v, want [a b]/[10 20]", left, right)
	}
}

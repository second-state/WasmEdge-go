package wasmedge

import (
	"fmt"
	"math"
)

func checkedUint32Count(subject string, count uint64) (uint32, error) {
	if count > math.MaxUint32 {
		return 0, fmt.Errorf(
			"%s count %d exceeds the native uint32 limit: %w",
			subject, count, ErrInvalidArgument,
		)
	}
	return uint32(count), nil
}

func checkedCIntCount(subject string, count uint64) (int32, error) {
	if count > math.MaxInt32 {
		return 0, fmt.Errorf(
			"%s count %d exceeds the native int limit: %w",
			subject, count, ErrInvalidArgument,
		)
	}
	return int32(count), nil
}

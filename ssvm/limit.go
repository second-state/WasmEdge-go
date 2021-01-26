package ssvm

// #include <ssvm.h>
import "C"

type Limit struct {
	min    uint
	max    uint
	hasmax bool
}

func NewLimit(vals ...interface{}) Limit {
	self := Limit{
		min:    0,
		max:    0,
		hasmax: false,
	}
	if len(vals) == 0 || len(vals) > 2 {
		panic("Wrong argument of NewLimit(), parameter length must be 0 or 1")
	}

	switch vals[0].(type) {
	case int, int32, int64, uint, uint32, uint64:
		self.min = vals[0].(uint)
	default:
		panic("Wrong argument of NewLimit(), parameter types must be integer")
	}

	if len(vals) == 2 {
		switch vals[0].(type) {
		case int, int32, int64, uint, uint32, uint64:
			self.max = vals[0].(uint)
			self.hasmax = true
		default:
			panic("Wrong argument of NewLimit(), parameter types must be integer")
		}
		if self.max < self.min {
			panic("Wrong argument of NewLimit(), max value should be greater or equal to min value")
		}
	}
	return self
}

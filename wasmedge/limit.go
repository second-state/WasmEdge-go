package wasmedge

// #include <wasmedge.h>
import "C"

type Limit struct {
	min    uint
	max    uint
	hasmax bool
}

func NewLimit(minVal uint) *Limit {
	l := &Limit{
		min:    minVal,
		hasmax: false,
	}
	return l
}

func (l *Limit) WithMaxVal(maxVal uint) *Limit {
	l.hasmax = true
	l.max = maxVal
	return l
}

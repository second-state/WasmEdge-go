package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type Limit struct {
	min    uint
	max    uint
	hasmax bool
}

func NewLimit(minVal uint) *Limit {
	l := &Limit{
		min:    minVal,
		max:    minVal,
		hasmax: false,
	}
	return l
}

func NewLimitWithMax(minVal uint, maxVal uint) *Limit {
	if maxVal >= minVal {
		return &Limit{
			min:    minVal,
			max:    maxVal,
			hasmax: true,
		}
	}
	return nil
}

func (l *Limit) HasMax() bool {
	return l.hasmax
}

func (l *Limit) GetMin() uint {
	return l.min
}

func (l *Limit) GetMax() uint {
	return l.max
}

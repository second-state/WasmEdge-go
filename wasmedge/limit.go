package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type Limit struct {
	min    uint
	max    uint
	hasmax bool
	shared bool
}

func NewLimit(minVal uint) *Limit {
	l := &Limit{
		min:    minVal,
		max:    minVal,
		hasmax: false,
		shared: false,
	}
	return l
}

func NewLimitWithMax(minVal uint, maxVal uint) *Limit {
	if maxVal >= minVal {
		return &Limit{
			min:    minVal,
			max:    maxVal,
			hasmax: true,
			shared: false,
		}
	}
	return nil
}

func NewLimitShared(minVal uint) *Limit {
	l := &Limit{
		min:    minVal,
		max:    minVal,
		hasmax: false,
		shared: true,
	}
	return l
}

func NewLimitSharedWithMax(minVal uint, maxVal uint) *Limit {
	if maxVal >= minVal {
		return &Limit{
			min:    minVal,
			max:    maxVal,
			hasmax: true,
			shared: true,
		}
	}
	return nil
}

func (l *Limit) HasMax() bool {
	return l.hasmax
}

func (l *Limit) IsShared() bool {
	return l.shared
}

func (l *Limit) GetMin() uint {
	return l.min
}

func (l *Limit) GetMax() uint {
	return l.max
}

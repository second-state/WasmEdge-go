package ssvm

// #include <ssvm.h>
import "C"

type Interpreter struct {
	_inner *C.SSVM_InterpreterContext
}

func NewInterpreter() *Interpreter {
	self := &Interpreter{
		_inner: C.SSVM_InterpreterCreate(nil, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewInterpreterWithConfig(conf *Configure) *Interpreter {
	self := &Interpreter{
		_inner: C.SSVM_InterpreterCreate(conf._inner, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewInterpreterWithStatistics(stat *Statistics) *Interpreter {
	self := &Interpreter{
		_inner: C.SSVM_InterpreterCreate(nil, stat._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewInterpreterWithConfigAndStatistics(conf *Configure, stat *Statistics) *Interpreter {
	self := &Interpreter{
		_inner: C.SSVM_InterpreterCreate(conf._inner, stat._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Interpreter) Delete() {
	C.SSVM_InterpreterDelete(self._inner)
	self._inner = nil
}

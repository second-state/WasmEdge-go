package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

type ValType struct {
	_inner C.WasmEdge_ValType
}

type ValMut C.enum_WasmEdge_Mutability

const (
	ValMut_Const = ValMut(C.WasmEdge_Mutability_Const)
	ValMut_Var   = ValMut(C.WasmEdge_Mutability_Var)
)

func NewValTypeI32() *ValType {
	return &ValType{_inner: C.WasmEdge_ValTypeGenI32()}
}

func NewValTypeI64() *ValType {
	return &ValType{_inner: C.WasmEdge_ValTypeGenI64()}
}

func NewValTypeF32() *ValType {
	return &ValType{_inner: C.WasmEdge_ValTypeGenF32()}
}

func NewValTypeF64() *ValType {
	return &ValType{_inner: C.WasmEdge_ValTypeGenF64()}
}

func NewValTypeV128() *ValType {
	return &ValType{_inner: C.WasmEdge_ValTypeGenV128()}
}

func NewValTypeFuncRef() *ValType {
	return &ValType{_inner: C.WasmEdge_ValTypeGenFuncRef()}
}

func NewValTypeExternRef() *ValType {
	return &ValType{_inner: C.WasmEdge_ValTypeGenExternRef()}
}

func (self *ValType) String() string {
	if C.WasmEdge_ValTypeIsI32(self._inner) {
		return "i32"
	}
	if C.WasmEdge_ValTypeIsI64(self._inner) {
		return "i64"
	}
	if C.WasmEdge_ValTypeIsF32(self._inner) {
		return "f32"
	}
	if C.WasmEdge_ValTypeIsF64(self._inner) {
		return "f64"
	}
	if C.WasmEdge_ValTypeIsV128(self._inner) {
		return "v128"
	}
	if C.WasmEdge_ValTypeIsFuncRef(self._inner) {
		return "funcref"
	}
	if C.WasmEdge_ValTypeIsExternRef(self._inner) {
		return "externref"
	}
	if C.WasmEdge_ValTypeIsRef(self._inner) {
		return "anyref"
	}
	panic("Unknown value type")
}

func (self *ValType) IsEqual(vt *ValType) bool {
	return bool(C.WasmEdge_ValTypeIsEqual(self._inner, vt._inner))
}

func (self *ValType) IsI32() bool {
	return bool(C.WasmEdge_ValTypeIsI32(self._inner))
}

func (self *ValType) IsI64() bool {
	return bool(C.WasmEdge_ValTypeIsI64(self._inner))
}

func (self *ValType) IsF32() bool {
	return bool(C.WasmEdge_ValTypeIsF32(self._inner))
}

func (self *ValType) IsF64() bool {
	return bool(C.WasmEdge_ValTypeIsF64(self._inner))
}

func (self *ValType) IsV128() bool {
	return bool(C.WasmEdge_ValTypeIsV128(self._inner))
}

func (self *ValType) IsFuncRef() bool {
	return bool(C.WasmEdge_ValTypeIsFuncRef(self._inner))
}

func (self *ValType) IsExternRef() bool {
	return bool(C.WasmEdge_ValTypeIsExternRef(self._inner))
}

func (self *ValType) IsRef() bool {
	return bool(C.WasmEdge_ValTypeIsRef(self._inner))
}

func (self *ValType) IsRefNull() bool {
	return bool(C.WasmEdge_ValTypeIsRefNull(self._inner))
}

func (self ValMut) String() string {
	switch self {
	case ValMut_Const:
		return "const"
	case ValMut_Var:
		return "var"
	}
	panic("Unknown value mutability")
}

type externRefManager struct {
	mu sync.Mutex
	// Valid next index of map. Use and increase this index when gc is empty.
	idx uint
	// Recycled entries of map. Use entry in this slide when allocate a new external reference.
	gc  []uint
	ref map[uint]interface{}
}

func (self *externRefManager) add(ptr interface{}) uint {
	self.mu.Lock()
	defer self.mu.Unlock()

	var realidx uint
	if len(self.gc) > 0 {
		realidx = self.gc[len(self.gc)-1]
		self.gc = self.gc[0 : len(self.gc)-1]
	} else {
		realidx = self.idx
		self.idx++
	}
	self.ref[realidx] = ptr
	return realidx
}

func (self *externRefManager) get(i uint) interface{} {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.ref[i]
}

func (self *externRefManager) del(i uint) {
	self.mu.Lock()
	defer self.mu.Unlock()
	delete(self.ref, i)
	self.gc = append(self.gc, i)
}

var externRefMgr = externRefManager{
	/// Index = 0 is reserved for ref.null
	idx: 1,
	ref: make(map[uint]interface{}),
}

type FuncRef struct {
	_inner C.WasmEdge_Value
}

func NewFuncRef(funcinst *Function) FuncRef {
	return FuncRef{
		_inner: C.WasmEdge_ValueGenFuncRef(funcinst._inner),
	}
}

func (self FuncRef) GetRef() *Function {
	funcinst := C.WasmEdge_ValueGetFuncRef(self._inner)
	if funcinst != nil {
		return &Function{_inner: funcinst, _own: false}
	}
	return nil
}

type ExternRef struct {
	_inner C.WasmEdge_Value
	_valid bool
}

func NewExternRef(ptr interface{}) ExternRef {
	// Gen an i64 WasmEdge_Value and change type to externref
	idx := uint64(externRefMgr.add(ptr))
	val := C.WasmEdge_ValueGenI64(C.int64_t(idx))
	val.Type = C.WasmEdge_ValTypeGenExternRef()
	return ExternRef{
		_inner: val,
		_valid: true,
	}
}

func (self ExternRef) Release() {
	self._valid = false
	// Change type back to WasmEdge_ValType_I64 and get the i64 value
	idx := uint(C.WasmEdge_ValueGetI64(self._inner))
	externRefMgr.del(idx)
}

func (self ExternRef) GetRef() interface{} {
	if self._valid {
		// Get the original i64 value
		idx := uint(C.WasmEdge_ValueGetI64(self._inner))
		return externRefMgr.get(idx)
	}
	return nil
}

type V128 struct {
	_inner C.WasmEdge_Value
}

func NewV128(high uint64, low uint64) V128 {
	var cval C.__int128
	var buf []byte
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&buf)))
	sliceHeader.Cap = 16
	sliceHeader.Len = 16
	sliceHeader.Data = uintptr(unsafe.Pointer(&cval))
	binary.LittleEndian.PutUint64(buf[:8], low)
	binary.LittleEndian.PutUint64(buf[8:], high)
	return V128{
		_inner: C.WasmEdge_ValueGenV128(cval),
	}
}

func (self V128) GetVal() (uint64, uint64) {
	cval := C.WasmEdge_ValueGetV128(self._inner)
	var buf []byte
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&buf)))
	sliceHeader.Cap = 16
	sliceHeader.Len = 16
	sliceHeader.Data = uintptr(unsafe.Pointer(&cval))
	return binary.LittleEndian.Uint64(buf[8:]), binary.LittleEndian.Uint64(buf[:8])
}

func toWasmEdgeValue(value interface{}) C.WasmEdge_Value {
	switch value.(type) {
	case FuncRef:
		return value.(FuncRef)._inner
	case ExternRef:
		if !value.(ExternRef)._valid {
			panic("External reference is released")
		}
		return value.(ExternRef)._inner
	case V128:
		return value.(V128)._inner
	case int:
		if unsafe.Sizeof(value.(int)) == 4 {
			return C.WasmEdge_ValueGenI32(C.int32_t(value.(int)))
		} else {
			return C.WasmEdge_ValueGenI64(C.int64_t(value.(int)))
		}
	case int32:
		return C.WasmEdge_ValueGenI32(C.int32_t(value.(int32)))
	case int64:
		return C.WasmEdge_ValueGenI64(C.int64_t(value.(int64)))
	case uint:
		if unsafe.Sizeof(value.(uint)) == 4 {
			return C.WasmEdge_ValueGenI32(C.int32_t(int32(value.(uint))))
		} else {
			return C.WasmEdge_ValueGenI64(C.int64_t(int64(value.(uint))))
		}
	case uint32:
		return C.WasmEdge_ValueGenI32(C.int32_t(int32(value.(uint32))))
	case uint64:
		return C.WasmEdge_ValueGenI64(C.int64_t(int64(value.(uint64))))
	case float32:
		return C.WasmEdge_ValueGenF32(C.float(value.(float32)))
	case float64:
		return C.WasmEdge_ValueGenF64(C.double(value.(float64)))
	default:
		panic("Wrong argument of toWasmEdgeValue()")
	}
}

func fromWasmEdgeValue(value C.WasmEdge_Value) interface{} {
	if C.WasmEdge_ValTypeIsI32(value.Type) {
		return int32(C.WasmEdge_ValueGetI32(value))
	}
	if C.WasmEdge_ValTypeIsI64(value.Type) {
		return int64(C.WasmEdge_ValueGetI64(value))
	}
	if C.WasmEdge_ValTypeIsF32(value.Type) {
		return float32(C.WasmEdge_ValueGetF32(value))
	}
	if C.WasmEdge_ValTypeIsF64(value.Type) {
		return float64(C.WasmEdge_ValueGetF64(value))
	}
	if C.WasmEdge_ValTypeIsV128(value.Type) {
		return V128{_inner: value}
	}
	if C.WasmEdge_ValTypeIsFuncRef(value.Type) {
		return FuncRef{_inner: value}
	}
	if C.WasmEdge_ValTypeIsExternRef(value.Type) {
		idx := uint(C.WasmEdge_ValueGetI64(value))
		if _, ok := externRefMgr.ref[idx]; ok {
			return ExternRef{_inner: value, _valid: true}
		}
		return ExternRef{_inner: value, _valid: false}
	}
	if C.WasmEdge_ValTypeIsRef(value.Type) {
		return "externref"
	}
	panic("Wrong argument of fromWasmEdgeValue()")
}

func toWasmEdgeValueSlide(vals ...interface{}) []C.WasmEdge_Value {
	cvals := make([]C.WasmEdge_Value, len(vals))
	for i, val := range vals {
		cvals[i] = toWasmEdgeValue(val)
	}
	return cvals
}

func toWasmEdgeValueSlideBindgen(vm *VM, rettype bindgen, modname *string, vals ...interface{}) []C.WasmEdge_Value {
	//cvals := make([]C.WasmEdge_Value, len(vals))
	cvals := []C.WasmEdge_Value{}
	if rettype == Bindgen_return_array {
		// Array result address = 8
		cvals = append(cvals, C.WasmEdge_ValueGenI32(C.int32_t(8)))
	} else if rettype == Bindgen_return_i64 {
		// wasm-bindgen magic: Set memory offset for i64 return value
		cvals = append(cvals, C.WasmEdge_ValueGenI32(C.int32_t(0)))
	}
	for _, val := range vals {
		switch t := val.(type) {
		case FuncRef:
			panic("toWasmEdgeValueSlideBindgen(): Not support FuncRef now")
		case ExternRef:
			panic("toWasmEdgeValueSlideBindgen(): Not support ExternRef now")
		case V128:
			panic("toWasmEdgeValueSlideBindgen(): Not support v128 now")
		case int32:
			cvals = append(cvals, C.WasmEdge_ValueGenI32(C.int32_t(val.(int32))))
		case uint32:
			cvals = append(cvals, C.WasmEdge_ValueGenI32(C.int32_t(val.(uint32))))
		case int64:
			vall := C.WasmEdge_ValueGenI32(C.int32_t(uint32(val.(int64))))
			valu := C.WasmEdge_ValueGenI32(C.int32_t(uint32(val.(int64) >> 32)))
			cvals = append(cvals, vall, valu)
		case uint64:
			vall := C.WasmEdge_ValueGenI32(C.int32_t(uint32(val.(uint64))))
			valu := C.WasmEdge_ValueGenI32(C.int32_t(uint32(val.(uint64) >> 32)))
			cvals = append(cvals, vall, valu)
		case int:
			panic("toWasmEdgeValueSlideBindgen(): Not support int now, please use int32 or int64 instead")
		case uint:
			panic("toWasmEdgeValueSlideBindgen(): Not support uint now, please use uint32 or uint64 instead")
		case float32:
			panic("toWasmEdgeValueSlideBindgen(): Not support float32 now")
		case float64:
			panic("toWasmEdgeValueSlideBindgen(): Not support float64 now")
		case []byte:
			// Call malloc function
			mallocsize := uint32(len(val.([]byte)))
			var rets []interface{}
			var err error = nil
			if modname == nil {
				rets, err = vm.Execute("__wbindgen_malloc", mallocsize)
			} else {
				rets, err = vm.ExecuteRegistered(*modname, "__wbindgen_malloc", mallocsize)
			}
			if err != nil {
				panic("toWasmEdgeValueSlideBindgen(): malloc failed")
			}
			if len(rets) <= 0 {
				panic("toWasmEdgeValueSlideBindgen(): malloc function signature unexpected")
			}
			argaddr := C.WasmEdge_ValueGenI32(C.int32_t(rets[0].(int32)))
			argsize := C.WasmEdge_ValueGenI32(C.int32_t(mallocsize))
			cvals = append(cvals, argaddr, argsize)
			// Set bytes
			var mod *Module = nil
			var mem *Memory = nil
			if modname == nil {
				mod = vm.GetActiveModule()
			} else {
				store := vm.GetStore()
				mod = store.FindModule(*modname)
			}
			if mod != nil {
				memnames := mod.ListMemory()
				if len(memnames) <= 0 {
					panic("toWasmEdgeValueSlideBindgen(): memory instance not found")
				}
				mem = mod.FindMemory(memnames[0])
				mem.SetData(val.([]byte), uint(rets[0].(int32)), uint(mallocsize))
			}
		default:
			errorString := fmt.Sprintf("Wrong argument of toWasmEdgeValueSlideBindgen(): %T not supported", t)
			panic(errorString)
		}
	}
	return cvals
}

func fromWasmEdgeValueSlide(cvals []C.WasmEdge_Value) []interface{} {
	if len(cvals) > 0 {
		vals := make([]interface{}, len(cvals))
		for i, cval := range cvals {
			vals[i] = fromWasmEdgeValue(cval)
		}
		return vals
	}
	return []interface{}{}
}

func fromWasmEdgeValueSlideBindgen(vm *VM, rettype bindgen, modname *string, cvals []C.WasmEdge_Value) (interface{}, error) {
	returns := fromWasmEdgeValueSlide(cvals)
	switch rettype {
	case Bindgen_return_void:
		return nil, nil
	case Bindgen_return_i32:
		if len(returns) <= 0 {
			panic("Expected return i32, but got empty")
		}
		return returns[0], nil
	case Bindgen_return_i64:
		// Get memory context
		var mod *Module = nil
		var mem *Memory = nil
		if modname == nil {
			mod = vm.GetActiveModule()
		} else {
			store := vm.GetStore()
			mod = store.FindModule(*modname)
		}
		if mod != nil {
			memnames := mod.ListMemory()
			if len(memnames) > 0 {
				mem = mod.FindMemory(memnames[0])
			}
		}
		// Get int64
		if mem == nil {
			panic("fromWasmEdgeValueSlideBindgen(): memory instance not found")
		}
		buf, err := mem.GetData(0, 8)
		if err != nil {
			return nil, err
		}
		var num int64 = 0
		for i, val := range buf {
			num += int64(val) << (i * 8)
		}
		return num, nil
	case Bindgen_return_array:
		// Get memory context
		var mod *Module = nil
		var mem *Memory = nil
		if modname == nil {
			mod = vm.GetActiveModule()
		} else {
			store := vm.GetStore()
			mod = store.FindModule(*modname)
		}
		if mod != nil {
			memnames := mod.ListMemory()
			if len(memnames) > 0 {
				mem = mod.FindMemory(memnames[0])
			}
		}
		// Get address and length (array result address = 8)
		if mem == nil {
			panic("fromWasmEdgeValueSlideBindgen(): memory instance not found")
		}
		buf, err := mem.GetData(8, 8)
		if err != nil {
			return nil, err
		}
		var num int64 = 0
		for i, val := range buf {
			num += int64(val) << (i * 8)
		}
		// Get bytes
		var arraddr = int32(num)
		var arrlen = int32(num >> 32)
		buf, err = mem.GetData(uint(arraddr), uint(arrlen))
		if err != nil {
			return nil, err
		}
		// Free array
		if modname == nil {
			_, err = vm.Execute("__wbindgen_free", arraddr, arrlen)
		} else {
			_, err = vm.ExecuteRegistered(*modname, "__wbindgen_free", arraddr, arrlen)
		}
		if err != nil {
			panic("fromWasmEdgeValueSlideBindgen(): malloc failed")
		}
		return buf, nil
	default:
		panic("Wrong expected return type")
	}
}

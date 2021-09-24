package wasmedge

// #include <wasmedge.h>
import "C"
import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

type ValType C.enum_WasmEdge_ValType
type RefType C.enum_WasmEdge_RefType
type ValMut C.enum_WasmEdge_Mutability

const (
	ValType_I32       = ValType(C.WasmEdge_ValType_I32)
	ValType_I64       = ValType(C.WasmEdge_ValType_I64)
	ValType_F32       = ValType(C.WasmEdge_ValType_F32)
	ValType_F64       = ValType(C.WasmEdge_ValType_F64)
	ValType_V128      = ValType(C.WasmEdge_ValType_V128)
	ValType_FuncRef   = ValType(C.WasmEdge_ValType_FuncRef)
	ValType_ExternRef = ValType(C.WasmEdge_ValType_ExternRef)
)

const (
	RefType_FuncRef   = RefType(C.WasmEdge_ValType_FuncRef)
	RefType_ExternRef = RefType(C.WasmEdge_ValType_ExternRef)
)

const (
	ValMut_Const = ValMut(C.WasmEdge_Mutability_Const)
	ValMut_Var   = ValMut(C.WasmEdge_Mutability_Var)
)

func (self ValType) String() string {
	switch self {
	case ValType_I32:
		return "i32"
	case ValType_I64:
		return "i64"
	case ValType_F32:
		return "f32"
	case ValType_F64:
		return "f64"
	case ValType_V128:
		return "v128"
	case ValType_FuncRef:
		return "funcref"
	case ValType_ExternRef:
		return "externref"
	}
	panic("Unknown value type")
}

func (self RefType) String() string {
	switch self {
	case RefType_FuncRef:
		return "funcref"
	case RefType_ExternRef:
		return "externref"
	}
	panic("Unknown reference type")
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

func NewFuncRef(funcidx uint) FuncRef {
	return FuncRef{
		_inner: C.WasmEdge_ValueGenFuncRef(C.uint32_t(funcidx)),
	}
}

type ExternRef struct {
	_inner C.WasmEdge_Value
	_valid bool
}

func NewExternRef(ptr interface{}) ExternRef {
	idx := uint64(externRefMgr.add(ptr))
	val := toWasmEdgeValue(idx)
	val.Type = C.WasmEdge_ValType_ExternRef
	return ExternRef{
		_inner: val,
		_valid: true,
	}
}

func (self ExternRef) Release() {
	self._valid = false
	idx := uint(fromWasmEdgeValue(self._inner, C.WasmEdge_ValType_I64).(int64))
	externRefMgr.del(idx)
}

func (self ExternRef) GetRef() interface{} {
	if self._valid {
		idx := uint(fromWasmEdgeValue(self._inner, C.WasmEdge_ValType_I64).(int64))
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

func fromWasmEdgeValue(value C.WasmEdge_Value, origtype C.enum_WasmEdge_ValType) interface{} {
	switch origtype {
	case C.WasmEdge_ValType_I32:
		return int32(C.WasmEdge_ValueGetI32(value))
	case C.WasmEdge_ValType_I64:
		return int64(C.WasmEdge_ValueGetI64(value))
	case C.WasmEdge_ValType_F32:
		return float32(C.WasmEdge_ValueGetF32(value))
	case C.WasmEdge_ValType_F64:
		return float64(C.WasmEdge_ValueGetF64(value))
	case C.WasmEdge_ValType_V128:
		return V128{_inner: value}
	case C.WasmEdge_ValType_FuncRef:
		return FuncRef{_inner: value}
	case C.WasmEdge_ValType_ExternRef:
		idx := uint(C.WasmEdge_ValueGetI64(value))
		if _, ok := externRefMgr.ref[idx]; ok {
			return ExternRef{_inner: value, _valid: true}
		}
		return ExternRef{_inner: value, _valid: false}
	default:
		panic("Wrong argument of fromWasmEdgeValue()")
	}
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
			if modname != nil {
				rets, err = vm.ExecuteRegistered(*modname, "__wbindgen_malloc", mallocsize)
			} else {
				rets, err = vm.Execute("__wbindgen_malloc", mallocsize)
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
			store := vm.GetStore()
			var memnames []string
			var mem *Memory = nil
			if modname != nil {
				memnames = store.ListMemoryRegistered(*modname)
			} else {
				memnames = store.ListMemory()
			}
			if len(memnames) <= 0 {
				panic("toWasmEdgeValueSlideBindgen(): memory instance not found")
			}
			if modname != nil {
				mem = store.FindMemoryRegistered(*modname, memnames[0])
			} else {
				mem = store.FindMemory(memnames[0])
			}
			mem.SetData(val.([]byte), uint(rets[0].(int32)), uint(mallocsize))
		default:
			errorString := fmt.Sprintf("Wrong argument of toWasmEdgeValueSlideBindgen(): %T not supported", t)
			panic(errorString)
		}
	}
	return cvals
}

func fromWasmEdgeValueSlide(cvals []C.WasmEdge_Value, types []C.enum_WasmEdge_ValType) []interface{} {
	if len(types) > 0 {
		vals := make([]interface{}, len(types))
		for i, cval := range cvals {
			vals[i] = fromWasmEdgeValue(cval, types[i])
		}
		return vals
	}
	return []interface{}{}
}

func fromWasmEdgeValueSlideBindgen(vm *VM, rettype bindgen, modname *string, cvals []C.WasmEdge_Value, types []C.enum_WasmEdge_ValType) (interface{}, error) {
	returns := fromWasmEdgeValueSlide(cvals, types)
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
		store := vm.GetStore()
		var memnames []string
		var mem *Memory = nil
		if modname != nil {
			memnames = store.ListMemoryRegistered(*modname)
		} else {
			memnames = store.ListMemory()
		}
		if len(memnames) <= 0 {
			panic("fromWasmEdgeValueSlideBindgen(): memory instance not found")
		}
		if modname != nil {
			mem = store.FindMemoryRegistered(*modname, memnames[0])
		} else {
			mem = store.FindMemory(memnames[0])
		}
		// Get int64
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
		store := vm.GetStore()
		var memnames []string
		var mem *Memory = nil
		if modname != nil {
			memnames = store.ListMemoryRegistered(*modname)
		} else {
			memnames = store.ListMemory()
		}
		if len(memnames) <= 0 {
			panic("fromWasmEdgeValueSlideBindgen(): memory instance not found")
		}
		if modname != nil {
			mem = store.FindMemoryRegistered(*modname, memnames[0])
		} else {
			mem = store.FindMemory(memnames[0])
		}
		// Get address and length (array result address = 8)
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
		if modname != nil {
			_, err = vm.ExecuteRegistered(*modname, "__wbindgen_free", arraddr, arrlen)
		} else {
			_, err = vm.Execute("__wbindgen_free", arraddr, arrlen)
		}
		if err != nil {
			panic("fromWasmEdgeValueSlideBindgen(): malloc failed")
		}
		return buf, nil
	default:
		panic("Wrong expected return type")
	}
}

package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type ExternType C.enum_WasmEdge_ExternalType

const (
	ExternType_Function = ExternType(C.WasmEdge_ExternalType_Function)
	ExternType_Table    = ExternType(C.WasmEdge_ExternalType_Table)
	ExternType_Memory   = ExternType(C.WasmEdge_ExternalType_Memory)
	ExternType_Global   = ExternType(C.WasmEdge_ExternalType_Global)
	ExternType_Tag      = ExternType(C.WasmEdge_ExternalType_Tag)
)

type AST struct {
	_inner *C.WasmEdge_ASTModuleContext
	_own   bool
}

type FunctionType struct {
	_inner *C.WasmEdge_FunctionTypeContext
	_own   bool
}

type TableType struct {
	_inner *C.WasmEdge_TableTypeContext
	_own   bool
}

type MemoryType struct {
	_inner *C.WasmEdge_MemoryTypeContext
	_own   bool
}

type TagType struct {
	_inner *C.WasmEdge_TagTypeContext
	_own   bool
}

type GlobalType struct {
	_inner *C.WasmEdge_GlobalTypeContext
	_own   bool
}

type ImportType struct {
	_inner *C.WasmEdge_ImportTypeContext
	_ast   *C.WasmEdge_ASTModuleContext
	_own   bool
}

type ExportType struct {
	_inner *C.WasmEdge_ExportTypeContext
	_ast   *C.WasmEdge_ASTModuleContext
	_own   bool
}

func (self *AST) ListImports() []*ImportType {
	if self._inner != nil {
		var imptype []*ImportType
		var cimptype []*C.WasmEdge_ImportTypeContext
		ltypes := C.WasmEdge_ASTModuleListImportsLength(self._inner)
		if uint(ltypes) > 0 {
			imptype = make([]*ImportType, uint(ltypes))
			cimptype = make([]*C.WasmEdge_ImportTypeContext, uint(ltypes))
			C.WasmEdge_ASTModuleListImports(self._inner, &(cimptype[0]), ltypes)
			for i, val := range cimptype {
				imptype[i] = &ImportType{}
				imptype[i]._inner = val
				imptype[i]._ast = self._inner
				imptype[i]._own = false
			}
			return imptype
		}
	}
	return nil
}

func (self *AST) ListExports() []*ExportType {
	if self._inner != nil {
		var exptype []*ExportType
		var cexptype []*C.WasmEdge_ExportTypeContext
		ltypes := C.WasmEdge_ASTModuleListExportsLength(self._inner)
		if uint(ltypes) > 0 {
			exptype = make([]*ExportType, uint(ltypes))
			cexptype = make([]*C.WasmEdge_ExportTypeContext, uint(ltypes))
			C.WasmEdge_ASTModuleListExports(self._inner, &(cexptype[0]), ltypes)
			for i, val := range cexptype {
				exptype[i] = &ExportType{}
				exptype[i]._inner = val
				exptype[i]._ast = self._inner
				exptype[i]._own = false
			}
			return exptype
		}
	}
	return nil
}

func (self *AST) Release() {
	if self._own {
		C.WasmEdge_ASTModuleDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}

func NewFunctionType(params []*ValType, returns []*ValType) *FunctionType {
	var cparams = make([]C.WasmEdge_ValType, len(params))
	var creturns = make([]C.WasmEdge_ValType, len(returns))
	for i, t := range params {
		cparams[i] = t._inner
	}
	for i, t := range returns {
		creturns[i] = t._inner
	}
	var ptrparams *C.WasmEdge_ValType = nil
	var ptrreturns *C.WasmEdge_ValType = nil
	if len(params) > 0 {
		ptrparams = &(cparams[0])
	}
	if len(returns) > 0 {
		ptrreturns = &(creturns[0])
	}
	ftype := C.WasmEdge_FunctionTypeCreate(
		ptrparams, C.uint32_t(len(params)),
		ptrreturns, C.uint32_t(len(returns)))
	if ftype == nil {
		return nil
	}
	return &FunctionType{_inner: ftype, _own: true}
}

func (self *FunctionType) GetParametersLength() uint {
	return uint(C.WasmEdge_FunctionTypeGetParametersLength(self._inner))
}

func (self *FunctionType) GetParameters() []*ValType {
	if self._inner != nil {
		var valtype []*ValType
		var cvaltype []C.WasmEdge_ValType
		ltypes := C.WasmEdge_FunctionTypeGetParametersLength(self._inner)
		if uint(ltypes) > 0 {
			valtype = make([]*ValType, uint(ltypes))
			cvaltype = make([]C.WasmEdge_ValType, uint(ltypes))
			C.WasmEdge_FunctionTypeGetParameters(self._inner, &(cvaltype[0]), ltypes)
			for i, val := range cvaltype {
				valtype[i]._inner = val
			}
			return valtype
		}
	}
	return nil
}

func (self *FunctionType) GetReturnsLength() uint {
	return uint(C.WasmEdge_FunctionTypeGetReturnsLength(self._inner))
}

func (self *FunctionType) GetReturns() []*ValType {
	if self._inner != nil {
		var valtype []*ValType
		var cvaltype []C.WasmEdge_ValType
		ltypes := C.WasmEdge_FunctionTypeGetReturnsLength(self._inner)
		if uint(ltypes) > 0 {
			valtype = make([]*ValType, uint(ltypes))
			cvaltype = make([]C.WasmEdge_ValType, uint(ltypes))
			C.WasmEdge_FunctionTypeGetReturns(self._inner, &(cvaltype[0]), ltypes)
			for i, val := range cvaltype {
				valtype[i]._inner = val
			}
			return valtype
		}
	}
	return nil
}

func (self *FunctionType) Release() {
	if self._own {
		C.WasmEdge_FunctionTypeDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}

func NewTableType(rtype *ValType, lim *Limit) *TableType {
	climit := C.WasmEdge_Limit{
		HasMax: C.bool(lim.hasmax),
		Shared: C.bool(lim.shared),
		Min:    C.uint32_t(lim.min),
		Max:    C.uint32_t(lim.max),
	}
	ttype := C.WasmEdge_TableTypeCreate(rtype._inner, climit)
	if ttype == nil {
		return nil
	}
	return &TableType{_inner: ttype, _own: true}
}

func (self *TableType) GetRefType() *ValType {
	return &ValType{_inner: C.WasmEdge_TableTypeGetRefType(self._inner)}
}

func (self *TableType) GetLimit() *Limit {
	if self._inner != nil {
		climit := C.WasmEdge_TableTypeGetLimit(self._inner)
		return &Limit{
			min:    uint(climit.Min),
			max:    uint(climit.Max),
			hasmax: bool(climit.HasMax),
			shared: bool(climit.Shared),
		}
	}
	return nil
}

func (self *TableType) Release() {
	if self._own {
		C.WasmEdge_TableTypeDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}

func NewMemoryType(lim *Limit) *MemoryType {
	climit := C.WasmEdge_Limit{
		HasMax: C.bool(lim.hasmax),
		Shared: C.bool(lim.shared),
		Min:    C.uint32_t(lim.min),
		Max:    C.uint32_t(lim.max),
	}
	mtype := C.WasmEdge_MemoryTypeCreate(climit)
	if mtype == nil {
		return nil
	}
	return &MemoryType{_inner: mtype, _own: true}
}

func (self *MemoryType) GetLimit() *Limit {
	if self._inner != nil {
		climit := C.WasmEdge_MemoryTypeGetLimit(self._inner)
		return &Limit{
			min:    uint(climit.Min),
			max:    uint(climit.Max),
			hasmax: bool(climit.HasMax),
			shared: bool(climit.Shared),
		}
	}
	return nil
}

func (self *MemoryType) Release() {
	if self._own {
		C.WasmEdge_MemoryTypeDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}

func (self *TagType) GetFunctionType() *FunctionType {
	ftype := C.WasmEdge_TagTypeGetFunctionType(self._inner)
	if ftype == nil {
		return nil
	}
	return &FunctionType{_inner: ftype, _own: false}
}

func NewGlobalType(vtype *ValType, vmut ValMut) *GlobalType {
	cvmut := C.enum_WasmEdge_Mutability(vmut)
	gtype := C.WasmEdge_GlobalTypeCreate(vtype._inner, cvmut)
	if gtype == nil {
		return nil
	}
	return &GlobalType{_inner: gtype, _own: true}
}

func (self *GlobalType) GetValType() *ValType {
	return &ValType{_inner: C.WasmEdge_GlobalTypeGetValType(self._inner)}
}

func (self *GlobalType) GetMutability() ValMut {
	return ValMut(C.WasmEdge_GlobalTypeGetMutability(self._inner))
}

func (self *GlobalType) Release() {
	if self._own {
		C.WasmEdge_GlobalTypeDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}

func (self *ImportType) GetExternalType() ExternType {
	return ExternType(C.WasmEdge_ImportTypeGetExternalType(self._inner))
}

func (self *ImportType) GetModuleName() string {
	return fromWasmEdgeString(C.WasmEdge_ImportTypeGetModuleName(self._inner))
}

func (self *ImportType) GetExternalName() string {
	return fromWasmEdgeString(C.WasmEdge_ImportTypeGetExternalName(self._inner))
}

func (self *ImportType) GetExternalValue() interface{} {
	if self._inner == nil {
		return nil
	}
	switch self.GetExternalType() {
	case ExternType_Function:
		return &FunctionType{
			_inner: C.WasmEdge_ImportTypeGetFunctionType(self._ast, self._inner),
			_own:   false,
		}
	case ExternType_Table:
		return &TableType{
			_inner: C.WasmEdge_ImportTypeGetTableType(self._ast, self._inner),
			_own:   false,
		}
	case ExternType_Memory:
		return &MemoryType{
			_inner: C.WasmEdge_ImportTypeGetMemoryType(self._ast, self._inner),
			_own:   false,
		}
	case ExternType_Global:
		return &GlobalType{
			_inner: C.WasmEdge_ImportTypeGetGlobalType(self._ast, self._inner),
			_own:   false,
		}
	case ExternType_Tag:
		return &TagType{
			_inner: C.WasmEdge_ImportTypeGetTagType(self._ast, self._inner),
			_own:   false,
		}
	}
	panic("Unknown external type")
}

func (self *ExportType) GetExternalType() ExternType {
	return ExternType(C.WasmEdge_ExportTypeGetExternalType(self._inner))
}

func (self *ExportType) GetExternalName() string {
	return fromWasmEdgeString(C.WasmEdge_ExportTypeGetExternalName(self._inner))
}

func (self *ExportType) GetExternalValue() interface{} {
	if self._inner == nil {
		return nil
	}
	switch self.GetExternalType() {
	case ExternType_Function:
		return &FunctionType{
			_inner: C.WasmEdge_ExportTypeGetFunctionType(self._ast, self._inner),
			_own:   false,
		}
	case ExternType_Table:
		return &TableType{
			_inner: C.WasmEdge_ExportTypeGetTableType(self._ast, self._inner),
			_own:   false,
		}
	case ExternType_Memory:
		return &MemoryType{
			_inner: C.WasmEdge_ExportTypeGetMemoryType(self._ast, self._inner),
			_own:   false,
		}
	case ExternType_Global:
		return &GlobalType{
			_inner: C.WasmEdge_ExportTypeGetGlobalType(self._ast, self._inner),
			_own:   false,
		}
	case ExternType_Tag:
		return &TagType{
			_inner: C.WasmEdge_ExportTypeGetTagType(self._ast, self._inner),
			_own:   false,
		}
	}
	panic("Unknown external type")
}

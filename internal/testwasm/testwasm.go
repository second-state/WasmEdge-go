// Package testwasm hand-assembles tiny WASM binaries for the binding's
// tests, so the repository needs no wat2wasm toolchain, no network access
// and no binary fixtures in git. Each builder returns a fresh slice.
//
// The encodings follow the WASM 1.0 binary format: a module is the 8-byte
// header followed by sections, each `id byte + uleb size + payload`, and
// most payloads start with a uleb element count ("vector").
package testwasm

// Value type codes.
const (
	tI32 = 0x7F
	tI64 = 0x7E
)

// Section ids.
const (
	secType   = 1
	secImport = 2
	secFunc   = 3
	secMemory = 5
	secExport = 7
	secCode   = 10
	secData   = 11
)

// Export kinds.
const (
	kindFunc   = 0x00
	kindMemory = 0x02
)

func header() []byte {
	return []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}
}

func uleb(v uint64) []byte {
	var out []byte
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		out = append(out, b)
		if v == 0 {
			return out
		}
	}
}

func cat(chunks ...[]byte) []byte {
	var out []byte
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}

// vec prefixes the element count, then concatenates the elements.
func vec(items ...[]byte) []byte {
	return cat(uleb(uint64(len(items))), cat(items...))
}

func section(id byte, payload []byte) []byte {
	return cat([]byte{id}, uleb(uint64(len(payload))), payload)
}

func name(s string) []byte {
	return cat(uleb(uint64(len(s))), []byte(s))
}

// funcType encodes `0x60 vec(params) vec(results)`.
func funcType(params, results []byte) []byte {
	p := make([][]byte, len(params))
	for i, t := range params {
		p[i] = []byte{t}
	}
	r := make([][]byte, len(results))
	for i, t := range results {
		r[i] = []byte{t}
	}
	return cat([]byte{0x60}, vec(p...), vec(r...))
}

func export(n string, kind byte, idx uint64) []byte {
	return cat(name(n), []byte{kind}, uleb(idx))
}

func importFunc(module, field string, typeIdx uint64) []byte {
	return cat(name(module), name(field), []byte{kindFunc}, uleb(typeIdx))
}

// body encodes one code entry: `uleb size, vec(locals), instrs, end`.
func body(instrs []byte) []byte {
	inner := cat(vec(), instrs, []byte{0x0B})
	return cat(uleb(uint64(len(inner))), inner)
}

// AddModule exports `add: (i32, i32) -> i32` returning the sum.
func AddModule() []byte {
	return cat(
		header(),
		section(secType, vec(funcType([]byte{tI32, tI32}, []byte{tI32}))),
		section(secFunc, vec(uleb(0))),
		section(secExport, vec(export("add", kindFunc, 0))),
		section(secCode, vec(body([]byte{
			0x20, 0x00, // local.get 0
			0x20, 0x01, // local.get 1
			0x6A, // i32.add
		}))),
	)
}

// FibModule exports `fib: (i32) -> i32` with fib(0)=fib(1)=1, so
// fib(10)=89 and fib(21)=17711 (matching the classic WasmEdge example).
func FibModule() []byte {
	return cat(
		header(),
		section(secType, vec(funcType([]byte{tI32}, []byte{tI32}))),
		section(secFunc, vec(uleb(0))),
		section(secExport, vec(export("fib", kindFunc, 0))),
		section(secCode, vec(body([]byte{
			0x20, 0x00, // local.get 0
			0x41, 0x02, // i32.const 2
			0x48,       // i32.lt_s
			0x04, 0x7F, // if (result i32)
			0x41, 0x01, // i32.const 1
			0x05,       // else
			0x20, 0x00, // local.get 0
			0x41, 0x01, // i32.const 1
			0x6B,       // i32.sub
			0x10, 0x00, // call 0
			0x20, 0x00, // local.get 0
			0x41, 0x02, // i32.const 2
			0x6B,       // i32.sub
			0x10, 0x00, // call 0
			0x6A, // i32.add
			0x0B, // end (if)
		}))),
	)
}

// HostCallModule imports `env.host_add: (i32, i32) -> i32` and exports
// `call_host: (i32, i32) -> i32` forwarding to it.
func HostCallModule() []byte {
	return cat(
		header(),
		section(secType, vec(funcType([]byte{tI32, tI32}, []byte{tI32}))),
		section(secImport, vec(importFunc("env", "host_add", 0))),
		section(secFunc, vec(uleb(0))),
		section(secExport, vec(export("call_host", kindFunc, 1))),
		section(secCode, vec(body([]byte{
			0x20, 0x00, // local.get 0
			0x20, 0x01, // local.get 1
			0x10, 0x00, // call 0 (the import)
		}))),
	)
}

// MemoryModule exports a 1-page memory as "mem" (with "hello" written at
// offset 0) and `load8: (i32) -> i32` returning the byte at the given
// address.
func MemoryModule() []byte {
	return cat(
		header(),
		section(secType, vec(funcType([]byte{tI32}, []byte{tI32}))),
		section(secFunc, vec(uleb(0))),
		section(secMemory, vec(cat([]byte{0x00}, uleb(1)))), // min 1, no max
		section(secExport, vec(
			export("mem", kindMemory, 0),
			export("load8", kindFunc, 0),
		)),
		section(secCode, vec(body([]byte{
			0x20, 0x00, // local.get 0
			0x2D, 0x00, 0x00, // i32.load8_u align=0 offset=0
		}))),
		section(secData, vec(cat(
			[]byte{0x00},             // active segment, memory 0
			[]byte{0x41, 0x00, 0x0B}, // offset expr: i32.const 0; end
			name("hello"),            // 5 bytes of data
		))),
	)
}

// LoopModule exports `run: () -> ()` that loops forever; used to test
// cancellation and timeouts.
func LoopModule() []byte {
	return cat(
		header(),
		section(secType, vec(funcType(nil, nil))),
		section(secFunc, vec(uleb(0))),
		section(secExport, vec(export("run", kindFunc, 0))),
		section(secCode, vec(body([]byte{
			0x03, 0x40, // loop (void)
			0x0C, 0x00, // br 0
			0x0B, // end (loop)
		}))),
	)
}

// ProcExitModule imports `wasi_snapshot_preview1.proc_exit: (i32) -> ()`
// and exports `_start: () -> ()` calling it with exit code 7.
func ProcExitModule() []byte {
	return cat(
		header(),
		section(secType, vec(
			funcType([]byte{tI32}, nil), // type 0: proc_exit
			funcType(nil, nil),          // type 1: _start
		)),
		section(secImport, vec(importFunc("wasi_snapshot_preview1", "proc_exit", 0))),
		section(secFunc, vec(uleb(1))),
		section(secExport, vec(export("_start", kindFunc, 1))),
		section(secCode, vec(body([]byte{
			0x41, 0x07, // i32.const 7
			0x10, 0x00, // call 0 (proc_exit)
		}))),
	)
}

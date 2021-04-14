(module
  (type $t0 (func (param externref i32)))
  (type $t1 (func (param i32) (result i32)))
  ;; import host function
  (import "host" "print_val_and_res" (func $f-host (type $t0)))
  ;; import from fibonacci.wasm : fib
  (import "wasm" "fib" (func $f-fib (type $t1)))
  (func $print_val_and_fib (type $t0) (param $p0 externref) (param $p1 i32)
    ;; param0: external reference to an object
    ;; param1: i32 to call fib
    local.get $p0
    local.get $p1
    call $f-fib
    call $f-host)
  (export "print_val_and_fib" (func $print_val_and_fib)))

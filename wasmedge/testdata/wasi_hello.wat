(module
  (import "wasi_snapshot_preview1" "fd_write"
    (func $fd_write (param i32 i32 i32 i32) (result i32)))
  (import "wasi_snapshot_preview1" "proc_exit"
    (func $proc_exit (param i32)))
  (memory (export "memory") 1)
  (data (i32.const 12) "hello\n")
  (func (export "_start")
    ;; iovec at address 0: buf=12, buf_len=6
    (i32.store (i32.const 0) (i32.const 12))
    (i32.store (i32.const 4) (i32.const 6))
    (drop (call $fd_write (i32.const 1) (i32.const 0) (i32.const 1) (i32.const 20)))
    (call $proc_exit (i32.const 21))))

(module
  (func (export "trap")
    (unreachable))
  (func (export "div") (param i32 i32) (result i32)
    (i32.div_s (local.get 0) (local.get 1))))

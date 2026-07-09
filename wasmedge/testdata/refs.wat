(module
  (func (export "extern_id") (param externref) (result externref)
    (local.get 0))
  (func (export "func_id") (param funcref) (result funcref)
    (local.get 0))
  (func (export "is_null_extern") (param externref) (result i32)
    (ref.is_null (local.get 0)))
  (func (export "v128_id") (param v128) (result v128)
    (local.get 0))
  (func (export "splat_add") (param i32 i32) (result i32)
    (i32x4.extract_lane 0
      (i32x4.add (i32x4.splat (local.get 0)) (i32x4.splat (local.get 1))))))

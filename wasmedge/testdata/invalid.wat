(module
  ;; Well-formed binary but fails validation: missing return value.
  ;; Assemble with: wat2wasm --no-check
  (func (export "bad") (result i32)))

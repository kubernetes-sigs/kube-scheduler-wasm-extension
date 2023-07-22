;; prescore_from_global lets us test the value range of status_code.
(module $prescore_from_global

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  ;; status_code is set by the host.
  (global $status_code (export "status_code_global") (mut i32) (i32.const 0))

  (func (export "prescore") (result i32) (return (global.get $status_code)))

  ;; We require exporting score with prescore
  (func (export "score") (result i64) (unreachable))
)

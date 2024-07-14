;; prefilterextensions_from_global lets us test the value range of status_code.
(module $prefilterextensions_from_global

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  ;; status_code is set by the host.
  (global $status_code (export "status_code_global") (mut i32) (i32.const 0))

  (func (export "addpod") (result i32) (return (global.get $status_code)))
  (func (export "removepod") (result i32) (return (global.get $status_code)))
  ;; We require exporting filter
  (func (export "filter") (result i32) (unreachable))
)

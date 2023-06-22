;; noop lets us test the overhead of the host calling no-op plugin functions.
(module $noop

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  (func (export "prefilter") (result i32) (return (i32.const 0)))
  (func (export "filter") (result i32) (return (i32.const 0)))
  (func (export "score") (result i64) (return (i64.const 0)))
)

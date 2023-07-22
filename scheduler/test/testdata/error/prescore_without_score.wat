(module $prescore_without_score

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  ;; Test the error of not also defining score.
  (func (export "prescore") (result i32) (unreachable))
)

;; score_from_global lets us test the value range of score and status_code
(module $score_from_global

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  ;; score is set by the host.
  (global $score (export "score_global") (mut i32) (i32.const 0))
  ;; status_code is set by the host.
  (global $status_code (export "status_code_global") (mut i32) (i32.const 0))

  (func (export "score") (result i64)
    ;; var score int32
    (local $score i32)

    ;; var status_code int32
    (local $status_code i32)

    ;; score = global.score
    (local.set $score (global.get $score))

    ;; status_code = global.status_code
    (local.set $status_code (global.get $status_code))

    ;; return uint64(score) << 32 | uint64(status_code)
    (return
      (i64.or
        (i64.shl (i64.extend_i32_u (local.get $score)) (i64.const 32))
        (i64.extend_i32_u (local.get $status_code)))))
)

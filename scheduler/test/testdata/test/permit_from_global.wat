;; permit_from_global lets us test the value range of status_code
(module $permit_from_global

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  ;; status_code is set by the host.
  (global $status_code (export "status_code_global") (mut i32) (i32.const 0))
  ;; timeout is set by the host.
  (global $timeout (export "timeout_global") (mut i32) (i32.const 0))

  (func (export "permit") (result i64)
    ;; var status_code int32
    (local $status_code i32)

    ;; var timeout int32
    (local $timeout i32)

    ;; status_code = global.status_code
    (local.set $status_code (global.get $status_code))

    ;; timeout = global.timeout
    (local.set $timeout (global.get $timeout))

    ;; return uint64(timeout) << 32 | uint64(status_code)
    (return
      (i64.or
        (i64.shl (i64.extend_i32_u (local.get $status_code)) (i64.const 32))
        (i64.extend_i32_u (local.get $timeout)))))
)

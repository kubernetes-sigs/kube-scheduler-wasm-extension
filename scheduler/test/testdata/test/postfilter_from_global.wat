;; postfilter_from_global lets us test the value range of nominating_mode and status_code
(module $postfilter_from_global

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  ;; nominating_mode is set by the host.
  (global $nominating_mode (export "nominating_mode_global") (mut i32) (i32.const 0))
  ;; status_code is set by the host.
  (global $status_code (export "status_code_global") (mut i32) (i32.const 0))

  (func (export "postfilter") (result i64)
    ;; var nominating_mode int32
    (local $nominating_mode i32)

    ;; var status_code int32
    (local $status_code i32)

    ;; nominating_mode = global.nominating_mode
    (local.set $nominating_mode (global.get $nominating_mode))

    ;; status_code = global.status_code
    (local.set $status_code (global.get $status_code))

    ;; return uint64(nominating_mode) << 32 | uint64(status_code)
    (return
      (i64.or
        (i64.shl (i64.extend_i32_u (local.get $nominating_mode)) (i64.const 32))
        (i64.extend_i32_u (local.get $status_code)))))

  ;; We require exporting filter with postfilter
  (func (export "filter") (result i32) (unreachable))
)

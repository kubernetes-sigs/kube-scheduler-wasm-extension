;; postbind_from_global lets us test the value range of status_code
(module $postbind_from_global

  ;; flag is set by the host.
  (global $flag (export "flag_global") (mut i32) (i32.const 0))

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  (func (export "postbind")
    (if (i32.eq (global.get $flag) (i32.const 1))
      (unreachable)
    )
  )

  ;; We require exporting bind with postbind
  (func (export "bind") (result i32) (unreachable)) 
)

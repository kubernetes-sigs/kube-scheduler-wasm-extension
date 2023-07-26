;; panic_on_start is a WASI command which issues an unreachable instruction
;; after writing an error to stdout. This simulates a panic in TinyGo.
(module $panic_on_start
  ;; Import the fd_write function from wasi, used in TinyGo for println.
  (import "wasi_snapshot_preview1" "fd_write"
    (func $wasi.fd_write (param $fd i32) (param $iovs i32) (param $iovs_len i32) (param $result.size i32) (result (;errno;) i32)))

  ;; Export the required functions for the plugin ABI
  (func (export "filter") (result i32) (i32.const 0))

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  ;; Pre-populate memory with the panic message, in iovec format
  (data (i32.const 0) "\08")    ;; iovs[0].offset
  (data (i32.const 4) "\06")    ;; iovs[0].length
  (data (i32.const 8) "panic!") ;; iovs[0]

  ;; On start, write "panic!" to stdout and crash.
  (func (export "_start")
    ;; Write the panic to stdout via its iovec [offset, len].
    (call $wasi.fd_write
      (i32.const 1) ;; stdout
      (i32.const 0) ;; where's the iovec
      (i32.const 1) ;; only one iovec
      (i32.const 0) ;; overwrite the iovec with the ignored result.
    )
    drop ;; ignore the errno returned

    ;; Issue the unreachable instruction instead of exiting normally
    (unreachable))
)

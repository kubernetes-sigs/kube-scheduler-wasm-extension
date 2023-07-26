;; $panic_on_get_config is a prefilter which issues an unreachable instruction
;; after writing config to stdout. This is a way to prove configuration got to
;; the guest.
(module $panic_on_get_config
  ;; Import the fd_write function from wasi, used in TinyGo for println.
  (import "wasi_snapshot_preview1" "fd_write"
    (func $wasi.fd_write (param $fd i32) (param $iovs i32) (param $iovs_len i32) (param $result.size i32) (result (;errno;) i32)))

  ;; get_config writes configuration from the host to memory if it exists and
  ;; isn't larger than $buf_limit. The result is its length in bytes.
  (import "k8s.io/scheduler" "get_config" (func $get_config
    (param $buf i32) (param $buf_limit i32)
    (result (; len ;) i32)))

  ;; Allocate the minimum amount of memory, 1 page (64KB).
  (memory (export "memory") 1 1)

  ;; config_limit is the max size config to read.
  (global $config_limit i32 (i32.const 1024))

  ;; On prefilter, write "panic!" to stdout and crash.
  (func (export "prefilter") (result i32)
    (local $config_len i32)

    ;; Write config to offset 8, which is the location where the data for
    ;; stdout begins
    (i32.store (i32.const 0) (i32.const 8)) ;; iovs[0].offset
    (local.set $config_len
      (call $get_config (i32.const 8) (global.get $config_limit))) ;; iovs[0]

    ;; if config_len > config_limit { panic }
    (if (i32.gt_u (local.get $config_len) (global.get $config_limit))
      (then unreachable))

    ;; Write the length of configuration read.
    (i32.store (i32.const 4) (local.get $config_len)) ;; iovs[0].length

    ;; Write the panic to stdout via its iovec [offset, len].
    (call $wasi.fd_write
      (i32.const 1) ;; stdout
      (i32.const 0) ;; where's the iovec
      (i32.const 1) ;; only one iovec
      (i32.const 0) ;; overwrite the iovec with the ignored result.
    )
    drop ;; ignore the errno returned

    ;; Issue the unreachable instruction instead of returning a code
    (unreachable))
)

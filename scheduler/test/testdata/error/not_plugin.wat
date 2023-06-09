(module $not_plugin
  ;; exports a memory but nothing else. This is invalid!
  (memory (export "memory") 1 1)
)

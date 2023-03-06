# sync

sync provides primitives for working with collections of go routines and their associated context and error handling.

- `sync/ctxsync`: context aware sync primitives.
- `sync/errgroup`: simplifies common patterns of goroutine use, in particular
making it straightforward to reliably wait on parallel or pipelined
goroutines, exiting either when the first error is encountered or waiting
for all goroutines to finish regardless of error outcome. Contexts are used
to control cancelation. It is modeled on golang.org/x/sync/errgroup and
other similar packages. It makes use of cloudeng.io/errors to simplify
collecting multiple errors. 
- `sync/syncsort`: concurrency aware data sorting, including a heap.
- `sync/synctestutil`: testing support, including testing for goroutine leaks.


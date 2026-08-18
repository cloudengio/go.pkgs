# Package [cloudeng.io/os/lockedfile](https://pkg.go.dev/cloudeng.io/os/lockedfile?tab=doc)

```go
import cloudeng.io/os/lockedfile
```

Package lockedfile creates and manipulates files whose contents should only
change atomically. This package is forked from the go compiler's internal
packages.

## Functions
### Func Read
```go
func Read(name string) ([]byte, error)
```
Read opens the named file with a read-lock and returns its contents.

### Func Transform
```go
func Transform(name string, t func([]byte) ([]byte, error)) (err error)
```
Transform invokes t with the result of reading the named file, with its lock
still held.

If t returns a nil error, Transform then writes the returned contents back
to the file, making a best effort to preserve existing contents on error.

t must not modify the slice passed to it.

### Func Write
```go
func Write(name string, content io.Reader, perm os.FileMode) (err error)
```
Write opens the named file (creating it with the given permissions if
needed), then write-locks it and overwrites it with the given content.



## Types
### Type File
```go
type File struct {
	// contains filtered or unexported fields
}
```
A File is a locked *os.File.

Closing the file releases the lock.

If the program exits while a file is locked, the operating system releases
the lock but may not do so promptly: callers must ensure that all locked
files are closed before exiting.

### Functions

```go
func Create(name string) (*File, error)
```
Create is like os.Create, but returns a write-locked file.


```go
func Edit(name string) (*File, error)
```
Edit creates the named file with mode 0666 (before umask), but does not
truncate existing contents.

If Edit succeeds, methods on the returned File can be used for I/O.
The associated file descriptor has mode O_RDWR and the file is write-locked.


```go
func Open(name string) (*File, error)
```
Open is like os.Open, but returns a read-locked file.


```go
func OpenFile(name string, flag int, perm os.FileMode) (*File, error)
```
OpenFile is like os.OpenFile, but returns a locked file. If flag includes
os.O_WRONLY or os.O_RDWR, the file is write-locked; otherwise, it is
read-locked.


```go
func TryEdit(name string) (*File, bool, error)
```
TryEdit is like Edit but non-blocking: it returns ok=false (with a nil
error) if the file is already locked by another process. Because the lock is
held for the lifetime of the process (released on Close or exit, including
on crash), TryEdit is well suited to single-instance enforcement.


```go
func TryOpenFile(name string, flag int, perm os.FileMode) (f *File, ok bool, err error)
```
TryOpenFile is like OpenFile but, instead of blocking, returns ok=false
(with a nil error) if the file is already locked by another process.
On ok=true the returned *File must be Closed to release the lock.



### Methods

```go
func (f *File) Close() error
```
Close unlocks and closes the underlying file.

Close may be called multiple times; all calls after the first will return a
non-nil error.




### Type Mutex
```go
type Mutex struct {
	Path string // The path to the well-known lock file. Must be non-empty.
	// contains filtered or unexported fields
}
```
A Mutex provides mutual exclusion within and across processes by locking
a well-known file. Such a file generally guards some other part of the
filesystem: for example, a Mutex file in a directory might guard access to
the entire tree rooted in that directory.

Mutex does not implement sync.Locker: unlike a sync.Mutex,
a lockedfile.Mutex can fail to lock (e.g. if there is a permission error in
the filesystem).

Like a sync.Mutex, a Mutex may be included as a field of a larger struct but
must not be copied after first use. The Path field must be set before first
use and must not be change thereafter.

### Functions

```go
func MutexAt(path string) *Mutex
```
MutexAt returns a new Mutex with Path set to the given non-empty path.



### Methods

```go
func (mu *Mutex) Lock() (unlock func(), err error)
```
Lock attempts to lock the Mutex.

If successful, Lock returns a non-nil unlock function: it is provided as a
return-value instead of a separate method to remind the caller to check the
accompanying error. (See https://golang.org/issue/20803.)


```go
func (mu *Mutex) RLock() (unlock func(), err error)
```
RLock attempts to lock the Mutex for read-only access.


```go
func (mu *Mutex) RLockCreate() (unlock func(), err error)
```
RLockCreate attempts to lock the Mutex for read-only access but will create
the lock file if one does not already exist.


```go
func (mu *Mutex) String() string
```


```go
func (mu *Mutex) TryLock() (unlock func(), ok bool, err error)
```
TryLock is like Lock but, instead of blocking, returns ok=false (with a nil
error and a nil unlock) if the Mutex is already locked by another process.

This makes a Mutex suitable for single-instance enforcement: the lock is
held until unlock is called or the process exits (including on crash),
so it never goes stale.








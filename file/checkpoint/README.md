# Package [cloudeng.io/file/checkpoint](https://pkg.go.dev/cloudeng.io/file/checkpoint?tab=doc)

```go
import cloudeng.io/file/checkpoint
```

Package checkpoint provides a mechanism for checkpointing the state of an
ongoing operation. An operation is defined as any application activity that
can be meaningfully broken into smaller steps and that can be resumed from
one of those steps. The record of the successful completion of each step is
recorded as a 'checkpoint'.

## Types
### Type Operation
```go
type Operation interface {
	// Checkpoint records the successful completion of a step in the
	// operation.
	Checkpoint(ctx context.Context, label string, data []byte) (id string, err error)

	// Latest reads the latest recorded checkpoint.
	Latest(ctx context.Context) ([]byte, error)

	// Complete removes all checkpoints since the operation is
	// deemed to be have comleted successfully.
	Complete(ctx context.Context) error

	// Load reads the checkpoint with the specified id, the id
	// must have been returned by an earlier call to Checkpoint.
	Load(ctx context.Context, id string) ([]byte, error)
}
```
Operation is the interface for checkpointing an operation.

### Functions

```go
func NewDirectoryOperation(dir string) (Operation, error)
```
NewDirectoryOperation returns an implementation of Operation that
uses a directory on the local file system to record checkpoints. This
implementation locks the directory using os.Lockedfile and rescans it on
each call to Checkpoint to determine the latest entry. Consequently it is
not well suited to very large numbers of checkpoints.








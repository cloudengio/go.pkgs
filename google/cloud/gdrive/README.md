# Package [cloudeng.io/google/cloud/gdrive](https://pkg.go.dev/cloudeng.io/google/cloud/gdrive?tab=doc)

```go
import cloudeng.io/google/cloud/gdrive
```

Package gdrive provides an implementation of largefile.Reader for Google
Drive.

## Constants
### DefaultLargeFileBlockSize
```go
DefaultLargeFileBlockSize = 1024 * 1024 * 64 // Default block size is 64 MiB.


```



## Functions
### Func GetFileID
```go
func GetFileID(ctx context.Context, srv *drive.Service, query string) (*drive.File, error)
```
GetFileID retrieves a file by its name and returns file metadata including
ID and name.

### Func GetWithFields
```go
func GetWithFields(ctx context.Context, srv *drive.Service, fileID string, fields ...googleapi.Field) (*drive.File, error)
```
GetWithFields retrieves a file by its ID and returns the file metadata with
specified fields.

### Func ServiceFromJSON
```go
func ServiceFromJSON(ctx context.Context, creds []byte, scopes ...string) (*drive.Service, error)
```



## Types
### Type DriveReader
```go
type DriveReader struct {
	// contains filtered or unexported fields
}
```
DriveReader implements largefile.Reader for Google Drive.

### Functions

```go
func NewReader(ctx context.Context, service *drive.Service, fileID string, opts ...Option) (*DriveReader, error)
```
NewReader creates a new largefile.Reader for a Google Drive file. It fetches
the file's metadata (name, size, md5 checksum) to initialize the reader.



### Methods

```go
func (dr *DriveReader) ContentLengthAndBlockSize() (int64, int)
```
ContentLengthAndBlockSize implements largefile.Reader.


```go
func (dr *DriveReader) Digest() digests.Hash
```
Digest implements largefile.Reader.


```go
func (dr *DriveReader) FileID() string
```


```go
func (dr *DriveReader) GetReader(ctx context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error)
```
GetReader implements largefile.Reader.


```go
func (dr *DriveReader) Name() string
```
Name implements largefile.Reader.




### Type Option
```go
type Option func(*options)
```
Option is used to configure a new DriveReader.

### Functions

```go
func WithBlockSize(size int) Option
```
WithBlockSize sets the preferred block size for downloads.








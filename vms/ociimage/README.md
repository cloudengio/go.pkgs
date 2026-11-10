# Package [cloudeng.io/vms/ociimage](https://pkg.go.dev/cloudeng.io/vms/ociimage?tab=doc)

```go
import cloudeng.io/vms/ociimage
```

Package ociimage parses and carries references to images in the Open
Container Initiative format, which is the format every VM and container
technology supported here uses: tart, docker and anything else that pulls an
image from a registry.

A Reference is the parse of a reference such as "linux-ci",
"linux-ci:latest", "ghcr.io/cirruslabs/ubuntu:latest" or
"mm-1:5001/linux-ci@sha256:...", split into the registry, the repository
path, the image name and the tag or digest that identifies the version.
Parsing is purely syntactic: a Reference records what the reference says,
and String returns it unchanged, so that the reference handed to a tool is
the one its user wrote.

Whether a reference is local is the one rule worth stating. A registry
is recognised by its host looking like one — it contains a "." or a ":",
or is exactly "localhost" — and anything else in the leading position is
part of the repository path. So "myorg/linux-ci" names no registry, and what
that means is the using tool's business, not this package's: tart resolves
such a reference in its own local store, whereas a docker-like tool resolves
it against a default registry of its own, see Reference.WithDefaultRegistry.
Reference.Remote says only that a registry was named explicitly, which is
the question "is there anywhere to pull this from".

Source is where an image comes from and how it is obtained, for a backend to
embed in its own configuration so that every backend spells these the same
way. Pulling itself is left to the backend, which is the only thing that
knows how: a backend that can pull implements Puller.

Nothing here validates a reference beyond its structure. An empty reference,
an empty path segment, a tag or digest that is not one at all: these are
rejected, since they are mistakes worth catching when the configuration
naming them is read. Whether the image exists, and whether its name obeys
the full grammar of the specification, is left to the tool that resolves it.

## Variables
### ErrInvalidReference
```go
ErrInvalidReference = errors.New("invalid OCI image reference")

```
ErrInvalidReference is returned by Parse for a reference it cannot make
sense of. Use errors.Is to test for it; the error itself says which
reference was rejected and why.



## Functions
### Func Pull
```go
func Pull(ctx context.Context, p Puller, ref Reference, opts PullOptions) error
```
Pull pulls ref with p, and does nothing at all for a reference that names no
registry: there is nowhere to pull such an image from, and it is the local
store that is expected to hold it already, see Reference.Remote.



## Types
### Type PullOptions
```go
type PullOptions struct {
	// Insecure allows the image to be pulled from a registry without
	// verifying its certificate.
	Insecure bool
	// Concurrency is the concurrency to pull with, or 0 to leave it to the
	// tool.
	Concurrency int
}
```
PullOptions are the options for a pull that every tool that can pull
understands. A tool that understands neither ignores both.


### Type Puller
```go
type Puller interface {
	Pull(ctx context.Context, ref Reference, opts PullOptions) error
}
```
Puller is implemented by something that can fetch an image into the local
store of whatever creates VMs or containers from it, such as a provider
backed by a tool with a pull command of its own.


### Type Reference
```go
type Reference struct {
	// Registry is the host the image is pulled from, with its port if it has
	// one, or empty if the reference names none, see Remote.
	Registry string
	// Repository is the path between the registry and the name, with no
	// leading or trailing "/", or empty if there is none: "cirruslabs" for
	// "ghcr.io/cirruslabs/ubuntu".
	Repository string
	// Name is the image name alone, which is what something named after the
	// image uses: a tart VM name and a docker container name may not contain
	// the "/" and ":" of a full reference.
	Name string
	// Tag is the tag, or empty if the reference has none. It is not defaulted
	// to "latest": what the reference says and what it implies are kept
	// apart, so that String round-trips.
	Tag string
	// Digest is the "algorithm:hex" digest, or empty if the reference has
	// none. A reference may carry a tag and a digest both, in which case it
	// is the digest that identifies the image.
	Digest string
}
```
Reference is a parsed OCI image reference. It is purely syntactic:
it records what the reference says rather than what a particular tool will
do with it, and String returns it unchanged. Every field is a string,
so a Reference is comparable and can be used as a map key.

The zero value is the empty reference, which no valid reference parses to,
see IsZero.

### Functions

```go
func Parse(s string) (Reference, error)
```
Parse parses an OCI image reference.

The leading segment is taken to be a registry if it looks like a host:
it contains a "." or a ":", or is exactly "localhost". Anything else is part
of the repository path, so "myorg/linux-ci" names no registry, see Remote.



### Methods

```go
func (r Reference) Immutable() bool
```
Immutable reports whether the reference can only ever resolve to one image,
which is true of a digest and of nothing else: any tag may be pushed over,
":latest" by habit and the rest by mistake.


```go
func (r Reference) IsZero() bool
```
IsZero reports whether the reference is the zero value, which is what a
Reference that was never parsed holds: no valid reference parses to it.


```go
func (r Reference) Path() string
```
Path returns the reference without its registry, tag or digest:
the repository path and the image name, which is what identifies the image
within whichever registry holds it.


```go
func (r Reference) Remote() bool
```
Remote reports whether the reference names a registry explicitly, and so has
somewhere to be pulled from.

A reference that names none is local to whichever tool resolves it:
tart looks for it in its own store, whereas a docker-like tool resolves it
against a default registry of its own, see WithDefaultRegistry. A reference
carrying a digest is remote whatever else it says, since a digest identifies
content in a registry rather than anything a local store names.


```go
func (r Reference) String() string
```
String returns the reference, which for a Reference obtained from Parse is
the string it was parsed from.


```go
func (r Reference) WithDefaultRegistry(host string) Reference
```
WithDefaultRegistry returns the reference with host as its registry if
it names none, and unchanged otherwise. It is how a tool that resolves a
reference against a registry of its own, as a docker-like tool resolves a
bare name against Docker Hub, says so.


```go
func (r Reference) WithDigest(digest string) Reference
```
WithDigest returns the reference with its digest replaced, or removed if
digest is empty. The digest is not validated, unlike one that Parse reads.


```go
func (r Reference) WithRegistry(host string) Reference
```
WithRegistry returns the reference with its registry replaced, or removed if
host is empty.


```go
func (r Reference) WithTag(tag string) Reference
```
WithTag returns the reference with its tag replaced, or removed if tag
is empty. The digest, if any, is left as it is; it is the digest that
identifies the image when a reference carries both.




### Type Source
```go
type Source struct {
	Image           string `yaml:"image" doc:"the OCI image reference to create VMs from, such as 'linux-ci', 'linux-ci:latest' or 'ghcr.io/cirruslabs/ubuntu:latest'."`
	PullRemoteOnRun bool   `yaml:"pull_remote_on_run" doc:"pull the image from its registry before creating each VM, so that a moving tag such as :latest is picked up without restarting. Ignored for a reference that names no registry, which has nothing to pull from."`
	Insecure        bool   `yaml:"pull_insecure" doc:"allow pulling the image from a registry without verifying its certificate."`
	Concurrency     int    `yaml:"pull_concurrency" doc:"the concurrency to pull the image with; 0 leaves it to the tool that pulls it."`
}
```
Source is where a backend's image comes from: which image it creates its VMs
or containers from, and how that image is obtained. It carries serialisation
tags so that a backend can embed it inline in its own configuration, rather
than restating these, and so that every backend spells them the same way.

It is named for what it describes rather than being another Config, so that
it can be embedded alongside the configuration of the tool that uses it:
a struct cannot embed both tartvm.Config and an ociimage.Config, since the
two would contribute the same field name.

### Methods

```go
func (s Source) PullOptions() PullOptions
```
PullOptions returns the configured options for a pull.


```go
func (s Source) Reference() (Reference, error)
```
Reference parses the configured image.


```go
func (s Source) ShouldPull(ref Reference) bool
```
ShouldPull reports whether ref should be pulled before it is used: the
configuration asks for it and the reference names a registry to pull from. A
local reference has nothing to pull from, so asking for it is not an error,
it simply does not apply.


```go
func (s Source) Validate() error
```
Validate reports a configuration that cannot be used: one that names no
image, or one whose image is not a reference. It is for a backend to call
when its own configuration is read, so that a mistake in a configuration
file is reported then rather than when the first VM fails to be created.
A backend that locates its own settings in its messages reports these itself
instead, see Reference.








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

Source is where an image comes from and how it is obtained, for a backend
to embed in its own configuration so that every backend spells these the
same way. Pulling itself is left to the backend, which is the only thing
that knows how: a backend that can pull implements Puller, and reads the
PullOption values it is given with ResolvePullOptions.

Verifier verifies the signatures of an image, as cosign and others store
them in its registry, without this package depending on any of their
libraries: the program that does supplies the implementation. A Policy says
which Signers are trusted, a key or a keyless identity. Verify checks what
does not depend on the toolkit and fails closed: a local image, which has
no registry to hold a signature, is never verified, and the verified result
must be pinned to a digest. PullVerified pulls that digest rather than the
tag, which can be moved between being verified and being pulled. A backend
that creates VMs from the pulled image should use the returned reference,
and from a local store only if it was pulled that way.

Nothing here validates a reference beyond its structure. An empty reference,
an empty path segment, a tag or digest that is not one at all: these are
rejected, since they are mistakes worth catching when the configuration
naming them is read. Whether the image exists, and whether its name obeys
the full grammar of the specification, is left to the tool that resolves it.

## Variables
### ErrNotVerified, ErrUnsigned
```go
// ErrNotVerified is returned, or wrapped, by a Verifier for an image that
// does not satisfy the policy, and by Verify for one that cannot be
// verified at all.
ErrNotVerified = errors.New("image signature not verified")
// ErrUnsigned is returned, or wrapped, along with ErrNotVerified for an
// image that has no signatures, as opposed to one whose signatures are
// not acceptable.
ErrUnsigned = errors.New("image is not signed")

```

### ErrInvalidReference
```go
ErrInvalidReference = errors.New("invalid OCI image reference")

```
ErrInvalidReference is returned by Parse for a reference it cannot make
sense of. Use errors.Is to test for it; the error itself says which
reference was rejected and why.



## Functions
### Func Pin
```go
func Pin(ctx context.Context, v Verifier, policy *Policy, source string) (string, error)
```
Pin returns the reference to create a VM or container from when it is
created from the image named source, for the backends that do so. If policy
is nil, no verification was asked for and source is returned as it is.
Otherwise the image is verified, see Verify, and the verified digest is
returned as a reference to it alone, without its tag, so that what is
created is exactly what was verified: an image that is unsigned, or not
signed as the policy requires, is an error and not to be used.

### Func Pull
```go
func Pull(ctx context.Context, p Puller, ref Reference, opts ...PullOption) error
```
Pull pulls ref with p, and does nothing at all for a reference that names no
registry: there is nowhere to pull such an image from, and it is the local
store that is expected to hold it already, see Reference.Remote.



## Types
### Type Policy
```go
type Policy struct {
	Signers []Signer `yaml:"signers" doc:"the signers that are trusted: an image is accepted if it carries a valid signature from any one of them."`
	// TransparencyLog requires that the signature is recorded in a
	// transparency log, such as Rekor.
	TransparencyLog bool `yaml:"require_transparency_log" doc:"require that the signature is recorded in a transparency log."`
	// Annotations must all be present, with these values, in the signature.
	Annotations map[string]string `yaml:"annotations" doc:"annotations that the signature must carry, with these values."`
}
```
Policy is what an image must satisfy to be accepted: it carries a valid
signature from at least one of the Signers, and the signature meets the
other requirements. It holds no more than a Verifier needs to decide,
so that it can be read from a configuration file, see Source.

### Methods

```go
func (p Policy) Validate() error
```
Validate reports a policy that cannot accept anything meaningful, which is
one that trusts no signer: a policy is never to be silently permissive.




### Type PullOption
```go
type PullOption func(*pullOptions)
```
PullOption sets one of the options for a pull, see Pull and Puller.

### Functions

```go
func WithPullConcurrency(concurrency int) PullOption
```
WithPullConcurrency sets the concurrency to pull with, leaving it to the
tool for anything less than one.


```go
func WithPullInsecure(insecure bool) PullOption
```
WithPullInsecure allows the image to be pulled from a registry without
verifying its certificate, which is for a private registry whose certificate
cannot be verified, or which serves plain HTTP.




### Type PullSettings
```go
type PullSettings struct {
	// Insecure allows the image to be pulled from a registry without
	// verifying its certificate.
	Insecure bool
	// Concurrency is the concurrency to pull with, or 0 to leave it to the
	// tool.
	Concurrency int
}
```
PullSettings is what a pull was asked for: the options that every tool
which can pull understands, and a tool that understands none of them
ignores them all. It is how a Puller reads the options it is given,
see ResolvePullOptions, and is only ever read: a pull is asked for with
PullOption values.

### Functions

```go
func ResolvePullOptions(opts ...PullOption) PullSettings
```
ResolvePullOptions resolves opts, which is how a Puller reads what the pull
it is carrying out was asked for.




### Type Puller
```go
type Puller interface {
	Pull(ctx context.Context, ref Reference, opts ...PullOption) error
}
```
Puller is implemented by something that can fetch an image into the local
store of whatever creates VMs or containers from it, such as a provider
backed by a tool with a pull command of its own. An implementation reads the
options it is given with ResolvePullOptions.


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


```go
func PullVerified(ctx context.Context, p Puller, v Verifier, ref Reference, policy Policy, opts ...PullOption) (Reference, error)
```
PullVerified verifies ref and, only if it is accepted, pulls the digest that
was verified, which is returned for the caller to create VMs or containers
from, see Verified.Reference. The tag is dropped from it, as Pin does, so
that the reference names the verified digest alone: not every tool accepts
one that carries both, and the digest is what identifies the image when it
does. Like Pull it needs a Puller, but not one that does nothing for a local
reference: that is an error, see Verify.



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




### Type Signature
```go
type Signature struct {
	// Key is the key that verified the signature, empty for a keyless one.
	Key string
	// Issuer and Subject are the certified identity of a keyless signature.
	Issuer, Subject string
	// Annotations are those the signature carries.
	Annotations map[string]string
	// Logged is true if the signature is recorded in a transparency log.
	Logged bool
}
```
Signature describes a signature that satisfied a policy.


### Type Signer
```go
type Signer struct {
	// Key identifies a public key. How it is spelled, such as a file, a URL
	// or a KMS URI, is for the Verifier to say.
	Key string `yaml:"key" doc:"the public key that signed the image, as a file, URL or KMS reference understood by the verifier."`
	// Issuer and IssuerRegexp are the OIDC issuer of a keyless identity, as an exact
	// string or a regular expression.
	Issuer       string `yaml:"issuer" doc:"the OIDC issuer of the identity that signed the image."`
	IssuerRegexp string `yaml:"issuer_regexp" doc:"a regular expression that the OIDC issuer must match, instead of issuer."`
	// Subject and SubjectRegexp are the subject of a keyless identity, such as an
	// email address or a workflow URL.
	Subject       string `yaml:"subject" doc:"the subject of the identity that signed the image, such as an email address or workflow URL."`
	SubjectRegexp string `yaml:"subject_regexp" doc:"a regular expression that the subject must match, instead of subject."`
}
```
Signer is a signer that a Policy trusts: either a public key or, for keyless
signing, the identity that was certified to sign. Exactly one of Key,
or the Issuer and Subject, is set.

### Methods

```go
func (s Signer) Keyless() bool
```
Keyless reports whether s is an identity rather than a key.


```go
func (s Signer) Validate() error
```
Validate reports a Signer that does not say who is trusted: both a key
and an identity, neither, an identity that lacks its issuer or subject,
or a regular expression that does not compile. Requiring both an issuer and
a subject, and not merely one, ensures that a signer is never "anyone that
the issuer certifies".




### Type Source
```go
type Source struct {
	Image           string `yaml:"image" doc:"the OCI image reference to create VMs from, such as 'linux-ci', 'linux-ci:latest' or 'ghcr.io/cirruslabs/ubuntu:latest'."`
	PullRemoteOnRun bool   `yaml:"pull_remote_on_run" doc:"pull the image from its registry before creating each VM, so that a moving tag such as :latest is picked up without restarting. Ignored for a reference that names no registry, which has nothing to pull from."`
	Insecure        bool   `yaml:"pull_insecure" doc:"allow pulling the image from a registry without verifying its certificate."`
	Concurrency     int    `yaml:"pull_concurrency" doc:"the concurrency to pull the image with; 0 leaves it to the tool that pulls it."`
	// Verify, if set, is the policy that the image's signature must satisfy,
	// see Verify and PullVerified.
	Verify *Policy `yaml:"verify" doc:"require the image to carry a signature that satisfies this policy before it is used."`
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
func (s Source) PullOptions() []PullOption
```
PullOptions returns the configured options for a pull, to be passed to Pull
or PullVerified.


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




### Type Verified
```go
type Verified struct {
	// Reference is the image that was verified, pinned to its digest. It is
	// what is to be pulled and used, rather than the reference that was
	// asked about: a tag can be moved between verifying it and using it, but
	// the digest that carries the signature cannot.
	Reference Reference
	// Signatures are those that satisfied the policy.
	Signatures []Signature
}
```
Verified is the result of a successful verification.

### Functions

```go
func Verify(ctx context.Context, v Verifier, ref Reference, policy Policy) (Verified, error)
```
Verify verifies ref with v against policy, after checking everything about
the request and the answer that does not depend on the toolkit: that there
is a verifier and a valid policy, that the image is remote since a local
one has no registry to hold a signature and so can never satisfy a policy,
and that the verified reference is the one asked about, pinned to a digest.




### Type Verifier
```go
type Verifier interface {
	Verify(ctx context.Context, ref Reference, policy Policy) (Verified, error)
}
```
Verifier verifies the signatures of images, in the form that cosign and
others store them in the registry that holds the image. It is implemented by
whatever depends on a signing toolkit, so that this package does not.

An implementation resolves the reference to a digest if it does not have
one, and verifies the signatures of that digest and no other. It returns an
error that satisfies errors.Is(err, ErrNotVerified) if the image does not
satisfy the policy, and may return any other error for a failure to carry
out the verification, such as a registry that cannot be reached. Either way
the image is not to be used.






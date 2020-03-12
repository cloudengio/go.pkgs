[![CircleCI](https://circleci.com/gh/cloudengio/go.pkg.svg?style=svg)](https://circleci.com/gh/cloudengio/go.pkg)

# go.pkg contains a set of broadly useful go packages.

It contains the following submodules, each of which can be imported and
versioned independently.

- [errors](errors/README.md): provides support for working with go errors post go 1.13.
- [sync](sync/README.md): provides easy to use patterns for working with goroutines and concurrency.
- [text](text/README.md): provides for support for operating on text/in-memory data.

# CI notes
- circecli is used for unit tests.
- github actions are used for linting.
  
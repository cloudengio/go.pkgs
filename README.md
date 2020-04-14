[![CircleCI](https://circleci.com/gh/cloudengio/go.pkgs.svg?style=svg)](https://circleci.com/gh/cloudengio/go.pkgs)

# go.pkgs contains a set of broadly useful go packages.

It contains the following submodules, each of which can be imported and
versioned independently.

- [cmdutil](cmdutil/README.md): provides support for building command line tools.
- [errors](errors/README.md): provides support for working with go errors post go 1.13.
- [path](path/README.md): provides support for working with paths and filenames, including cloud storage systems.
- [sync](sync/README.md): provides easy to use patterns for working with goroutines and concurrency.
- [text](text/README.md): provides support for operating on text/in-memory data.

# CI notes
- circecli is used for unit tests.
- github actions are used for linting.
  

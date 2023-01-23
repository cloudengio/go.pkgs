![GithubActions](https://github.com/cosnicolaou/pbzip2/actions/workflows/linux.yml/badge.svg)
![GithubActions](https://github.com/cosnicolaou/pbzip2/actions/workflows/macos.yml/badge.svg)
![GithubActions](https://github.com/cosnicolaou/pbzip2/actions/workflows/windows.yml/badge.svg
![GithubActions](https://github.com/cosnicolaou/pbzip2/actions/workflows/lint-govuln.yml/badge.svg
![GithubActions](https://github.com/cosnicolaou/pbzip2/actions/workflows/codeql.yml/badge.svg)

# go.pkgs contains a set of broadly useful go packages.

It contains the following submodules, each of which can be imported and
versioned independently. 

- [algo](algo/README.md): common algorithm implementations.
- aws: Amazon Web Services related packages.
- [cmdutil](cmdutil/README.md): support for building command line tools.
- debug: support for instrumenting code to trace execution and communication.
- [errors](errors/README.md): support for working with go errors post go 1.13.
- [file](file/README.md): support for working with files and filesystems, including cloud storage systems.
- io: I/O related packages.
- [os](os/README.md): os related and/or specific packages.
- [path](path/README.md): support for working with paths and filenames, including cloud storage systems.
- [sync](sync/README.md): easy to use patterns for working with goroutines and concurrency.
- [text](text/README.md): support for operating on text/in-memory data.
- [webapp](webapp/README.md): support for implementing webapps.
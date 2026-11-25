# Package [cloudeng.io/cmdutil/subcmd/shells](https://pkg.go.dev/cloudeng.io/cmdutil/subcmd/shells?tab=doc)

```go
import cloudeng.io/cmdutil/subcmd/shells
```

Package shells generates bash and zsh shell completion scripts from a
cloudeng.io/cmdutil/subcmd command tree.

Users of subcmd can wire a completions subcommand directly into a YAML
command spec using Runner and Flags:

    cmdSet.Set("completions").MustRunner(
        shells.Runner(cmdSet.CommandSet, nil),
        &shells.Flags{},
    )

Or construct one programmatically with NewCommand.

## Constants
### BashShell, ZshShell
```go
const (
	// Bash is the shell name for bash completion scripts.
	BashShell = "bash"
	// ZshShell is the shell name for zsh completion scripts.
	ZshShell = "zsh"
)
```



## Functions
### Func Bash
```go
func Bash(w io.Writer, prog string, cmds *subcmd.CommandSet) error
```
Bash writes a bash completion script for cmds to w. If prog is empty,
filepath.Base(os.Args[0]) is used.

### Func Generate
```go
func Generate(w io.Writer, shell, prog string, cmds *subcmd.CommandSet) error
```
Generate writes a shell completion script for cmds to w. If prog is empty,
filepath.Base(os.Args[0]) is used. Supported values for shell are "bash" and
"zsh".

### Func NewCommand
```go
func NewCommand(name string, cmds *subcmd.CommandSet, out io.Writer) *subcmd.Command
```
NewCommand constructs a subcmd.Command named name that generates shell
completions for cmds and writes them to out (or os.Stdout if out is nil).

### Func Runner
```go
func Runner(cmds *subcmd.CommandSet, out io.Writer) subcmd.Runner
```
Runner returns a subcmd.Runner that writes a completion script for the shell
named in args[0] ("bash" or "zsh") to out (or os.Stdout if out is nil).
The command should accept one argument (<shell>) and use &Flags{} as its
flag struct.

### Func SupportedShells
```go
func SupportedShells() []string
```
SupportedShells returns the names of the shells supported by this package.

### Func Zsh
```go
func Zsh(w io.Writer, prog string, cmds *subcmd.CommandSet) error
```
Zsh writes a zsh completion script for cmds to w. If prog is empty,
filepath.Base(os.Args[0]) is used.



## Types
### Type Flags
```go
type Flags struct {
	Program string `subcmd:"program,,program name to generate completions for (defaults to the executable name)"`
}
```
Flags defines the flags accepted by the subcommand runner returned by
Runner.






# Package [cloudeng.io/cmdutil/cmdexec](https://pkg.go.dev/cloudeng.io/cmdutil/cmdexec?tab=doc)

```go
import cloudeng.io/cmdutil/cmdexec
```

Package cmdexec provides a means of executing multiple subcommands with
the ability to expand the command line arguments using Go's text/template
package and environment variables.

## Functions
### Func AppendToOSEnv
```go
func AppendToOSEnv(v ...string) []string
```
AppendToOSEnv returns a copy of os.Environ with the supplied environment
variables appended to it.



## Types
### Type Option
```go
type Option func(*options)
```
Option represents an option to New.

### Functions

```go
func WithCommandsPrefix(v ...string) Option
```
WithCommandsPrefix sets a common set of arguments prepended to all commands.


```go
func WithDryRun(v bool) Option
```
WithDryRun logs the commands that would be executed but does not actually
execute them.


```go
func WithEnv(v []string) Option
```
WithEnv sets the environment variables to be made available to the executed
command.


```go
func WithExpandMapping(v func(string) string) Option
```
WithExpandMapping sets the mapping function to be used to expand environment
variables in the command line arguments. The default is os.Getenv.


```go
func WithLogger(v func(string, ...any) (int, error)) Option
```
WithLogger sets the logger function to be used to log the verbose and dry
run output. The default is fmt.Printf.


```go
func WithStderr(v io.Writer) Option
```
WithStderr sets the writer to which the standard error of the commands will
be written.


```go
func WithStdout(v io.Writer) Option
```
WithStdout sets the writer to which the standard output of the commands will
be written.


```go
func WithTemplateFuncs(v template.FuncMap) Option
```
WithTemplateFuncs sets the template functions to be used to expand the
command line arguments.


```go
func WithTemplateVars(v any) Option
```
WithTemplateVars sets the template variables to be used to expand the
command line arguments.


```go
func WithVerbose(v bool) Option
```
WithVerbose sets the verbose flag for the commands which generally results
in the expanded command line execeuted being logged.


```go
func WithWorkingDir(v string) Option
```
WithWorkingDir sets the working directory for the commands.




### Type Runner
```go
type Runner struct {
	// contains filtered or unexported fields
}
```
Runner represents a command Runner

### Functions

```go
func New(name string, opts ...Option) *Runner
```
New creates a new Runner instance with the supplied name and options.



### Methods

```go
func (r *Runner) ExpandCommandLine(args ...string) ([]string, error)
```
ExpandCommandLine expands the supplied command line arguments using the
supplied template functions and template variables, followed by environment
variale expansion. Template expansion is performed before environment
variable expansion, that is, a template expression may evaulate to an
environment variable expression (eg. to return ${MYVAR}). NOTE that the
environment variables are expanded by the current process and not by the
executed command.


```go
func (r *Runner) Run(ctx context.Context, cmds ...string) error
```
Run executes the supplied commands with the expanded command line as per
ExpandCommandLine.






## Examples
### [ExampleRunner](https://pkg.go.dev/cloudeng.io/cmdutil/cmdexec?tab=doc#example-Runner)





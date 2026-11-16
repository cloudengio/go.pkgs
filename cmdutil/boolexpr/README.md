# Package [cloudeng.io/cmdutil/boolexpr](https://pkg.go.dev/cloudeng.io/cmdutil/boolexpr?tab=doc)

```go
import cloudeng.io/cmdutil/boolexpr
```

Package boolexpr provides a boolean expression evaluator and parser.
The supported operators are &&, || and ! (negation), and grouping via ().
The set of operands is defined by clients of the package by implementing
the Operand interface. Operands represent simple predicates against
which the value supplied to the expression is evaluated, as such, they
implicitly contain a value of their own that is assigned when the operand
is instantiated. For example, a simple string comparison operand would be
represented as "name='foo' || name='bar'" which evaluated to true if the
expression is evaluated for "foo" or "bar", but not otherwise.

## Types
### Type Item
```go
type Item struct {
	// contains filtered or unexported fields
}
```
Item represents an operator or operand in an expression. It is exposed to
allow client packages to create their own parsers.

### Functions

```go
func AND() Item
```
AND returns an AND item.


```go
func LeftBracket() Item
```
LeftBracket returns a left bracket item.


```go
func NOT() Item
```
NOT returns a NOT item.


```go
func NewOperandItem(op Operand) Item
```
NewOperandItem returns an item representing an operand.


```go
func OR() Item
```
OR returns an OR item.


```go
func RightBracket() Item
```
RightBracket returns a right bracket item.



### Methods

```go
func (it Item) String() string
```




### Type Operand
```go
type Operand interface {
	// Prepare is used to prepare the operand for evaluation, for example, to
	// compile a regular expression. Document and String must be callable before
	// Prepare is called. Eval and Needs must only be called after Prepare.
	Prepare() (Operand, error)

	// Eval must return false for any type that it does not support.
	Eval(any) bool

	// Needs returns true if the operand needs the specified type.
	Needs(reflect.Type) bool

	// Document returns a string documenting the operand.
	Document() string

	// String returns a string representation of the operand and its current value.
	String() string
}
```
Operand represents an operand. It is exposed to allow clients packages to
define custom operands.


### Type Parser
```go
type Parser struct {
	// contains filtered or unexported fields
}
```
Parser parses text boolean expressions using registered Operand factories.

### Functions

```go
func NewParser() *Parser
```
NewParser returns a new Parser with no registered operands.



### Methods

```go
func (p *Parser) HasOperand(name string) bool
```
HasOperand returns true if an operand factory is registered under name.


```go
func (p *Parser) ListOperands() []Operand
```
ListOperands returns the list of registered operands in alphanumeric order.


```go
func (p *Parser) Parse(input string) (T, error)
```
Parse parses the supplied input into a boolexpr.T. The supported operators
are ! (negation), && (and), || (or), with standard precedence (! > && > ||)
and grouping via (). Operands are represented as <operand>=<value> where the
value is interpreted by the operand. The <value> may be raw single-quoted
('...'), double-quoted ("..." with \ escapes), or unquoted with \-escaped
runes. An empty or whitespace-only input returns an empty T whose Eval
returns false. The set of available operands is those registered with the
parser before Parse is called.


```go
func (p *Parser) RegisterOperand(name string, factory func(name, value string) Operand)
```
RegisterOperand registers an operand factory under name. The factory must
tolerate an empty value when called by ListOperands for documentation.


```go
func (p *Parser) RemoveOperand(name string)
```
RemoveOperand removes any operand factory registered under name.




### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T represents a compiled boolean expression of Operand predicates evaluated
against a single input value using standard precedence (! > && > ||) and
short-circuit evaluation.

### Functions

```go
func New(items ...Item) (T, error)
```
New returns a compiled T built from the supplied items. Operator precedence
is ! (highest), then &&, then ||, with () for explicit grouping.



### Methods

```go
func (m T) Eval(v any) bool
```
Eval evaluates the expression against v using short-circuit evaluation.
An empty, default T always returns false.


```go
func (m T) IsEmpty() bool
```
IsEmpty returns true if the expression contains no operands or operators.


```go
func (m T) Needs(typ any) bool
```
Needs returns true if any operand in the expression needs the type of typ.


```go
func (m T) NeedsFor[V any]() bool
```
NeedsFor returns true if any operand in the expression needs type V.


```go
func (m T) NeedsType(t reflect.Type) bool
```
NeedsType returns true if any operand in the expression needs t.


```go
func (m T) String() string
```








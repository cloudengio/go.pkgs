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
allow clients packages to create their own parsers.

### Functions

```go
func AND() Item
```
And returns an AND item.


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

### Functions

```go
func NewParser() *Parser
```



### Methods

```go
func (p *Parser) ListOperands() []Operand
```
ListOperands returns the list of registered operands in alphanumeric order.


```go
func (p *Parser) Parse(input string) (T, error)
```
Parse parses the supplied input into a boolexpr.T. The supported syntax
is a boolean expression with and (&&), or (||) and grouping, via ().
Operands are represented as <operand>=<value> where the value is interpreted
by the operand. The <value> may be quoted using single-quotes or contain
escaped runes via \. The set of available operands is those registered with
the parser before Parse is called.


```go
func (p *Parser) RegisterOperand(name string, factory func(name, value string) Operand)
```


```go
func (p *Parser) RemoveOperand(name string)
```




### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T represents a boolean expression of regular expressions, file type and mod
time comparisons. It is evaluated against a single input value.

### Functions

```go
func New(items ...Item) (T, error)
```
New returns a new matcher.T built from the supplied items.



### Methods

```go
func (m T) Eval(v any) bool
```
Eval evaluates the matcher against the supplied value. An empty, default
matcher will always return false.


```go
func (m T) Needs(typ any) bool
```
HasOperand returns true if the matcher's expression contains an instance of
the specified operand.


```go
func (m T) String() string
```








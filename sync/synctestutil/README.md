# Package [cloudeng.io/sync/synctestutil](https://pkg.go.dev/cloudeng.io/sync/synctestutil?tab=doc)

```go
import cloudeng.io/sync/synctestutil
```


## Functions
### Func AssertNoGoroutines
```go
func AssertNoGoroutines(t Errorf) func()
```
AssertNoGoroutines is used to detect goroutine leaks.

Usage is as shown below:

    func TestExample(t *testing.T) {
    	defer synctestutil.AssertNoGoroutines(t, time.Second)()
    	...
    }

Note that in the example above AssertNoGoroutines returns a function that is
immediately defered. The call to AssertNoGoroutines records the currently
goroutines and the returned function will compare that initial set to those
running when it is invoked. Hence, the above example is equivalent to:

    func TestExample(t *testing.T) {
    	fn := synctestutil.AssertNoGoroutines(t, time.Second)
    	...
    	fn()
    }

### Func AssertNoGoroutinesRacy
```go
func AssertNoGoroutinesRacy(t Errorf, wait time.Duration) func()
```
AssertNoGoroutinesRacy is like AssertNoGoroutines but allows for a grace
period for goroutines to terminate.



## Types
### Type Errorf
```go
type Errorf interface {
	Errorf(format string, args ...any)
}
```
Errorf is called when an error is encountered and is defined so that
testing.T and testing.B implement Errorf.






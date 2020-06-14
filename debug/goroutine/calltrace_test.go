package goroutine_test

import (
	"fmt"
	"sync"

	"cloudeng.io/debug/goroutine"
)

func ExampleCallTrace() {
	ct := &goroutine.CallTrace{}
	ct.Record("a")
	ct.Record("b")
	var wg sync.WaitGroup
	n := 2
	wg.Add(n)
	for i := 0; i < n; i++ {
		ct := ct.Go("goroutine launch")
		go func(i int) {
			ct.Record(fmt.Sprintf("inside goroutine %v", i))
			wg.Done()
			ct.Record(fmt.Sprintf("inside goroutine %v", i))
		}(i)
	}
	wg.Wait()
	fmt.Println(ct.String())
	fmt.Println(ct.DebugString())
	// output:
	// xx yy
}

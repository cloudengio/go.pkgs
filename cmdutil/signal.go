package cmdutil

import (
	"fmt"
	"os"
	"os/signal"
)

// HandleSignal will asynchronously invoke the supplied function when the specified signals
// are received.
func HandleSignals(fn func(), signals ...os.Signal) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, signals...)
	go func() {
		sig := <-sigCh
		fmt.Println("stopping on... ", sig)
		fn()
	}()
}

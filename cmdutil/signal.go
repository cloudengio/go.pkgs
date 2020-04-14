package cmdutil

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
)

// HandleSignal will asynchronously invoke the supplied function when the
// specified signals are received.
func HandleSignals(fn func(), signals ...os.Signal) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, signals...)
	go func() {
		sig := <-sigCh
		fmt.Println("stopping on... ", sig)
		fn()
	}()
}

// Exit formats and prints the supplied parameters to os.Stderr and then
// calls os.Exit(1).
func Exit(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

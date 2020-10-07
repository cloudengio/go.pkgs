package main

import (
	"context"
	"os"
)

type eraseFlags struct {
	CommonFlags
}

func erase(ctx context.Context, values interface{}, args []string) error {
	flagValues := values.(*eraseFlags)
	return os.RemoveAll(flagValues.DatabaseDir)
}

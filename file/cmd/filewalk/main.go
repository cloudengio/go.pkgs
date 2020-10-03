package main

import (
	"context"
	"fmt"

	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/file/filewalk"
)

type walkFlags struct {
	Concurrency int `subcmd:concurrency,-1,number of threads to use for scanning"`
}

var cmdSet *subcmd.CommandSet

func init() {
	fs := subcmd.NewFlagSet()
	fs.MustRegisterFlagStruct(&walkFlags{}, nil, nil)
	duCmd := subcmd.NewCommand("du", fs, du)
	duCmd.Document("gather file number and size statistics", "<prefix>+")
	cmdSet = subcmd.NewCommandSet(duCmd)
}

func main() {
	ctx := context.Background()
	cmdSet.Dispatch(ctx)
}

func fileFn(ctx context.Context, prefix string, ch <-chan filewalk.Contents) error {
	for results := range ch {
		fmt.Printf("file: %s: %v\n", prefix, results.Path)
	}
	return nil
}

func prefixFn(ctx context.Context, prefix string, info *filewalk.Info, err error) (bool, error) {
	fmt.Printf("prefix: %v\n", prefix)
	return false, nil
}

func du(ctx context.Context, values interface{}, args []string) error {
	flagValues := values.(*walkFlags)
	sc := filewalk.LocalScanner()
	walker := filewalk.New(sc, filewalk.Concurrency(flagValues.Concurrency))
	return walker.Walk(ctx, prefixFn, fileFn, args...)
}

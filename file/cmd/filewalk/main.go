package main

import (
	"context"

	"cloudeng.io/cmdutil/subcmd"
)

type walkFlags struct {
	Config string `subcmd:"config,.filewalk.yml,config file"`
}

var cmdSet *subcmd.CommandSet

func init() {
	fs := subcmd.NewFlagSet()
	fs.MustRegisterFlagStruct(&walkFlags{}, nil, nil)
	duCmd := subcmd.NewCommand("du", fs, du)
	duCmd.Document("gather file number and size statistics")
	cmdSet = subcmd.NewCommandSet(duCmd)
}

func main() {
	ctx := context.Background()
	cmdSet.Dispatch(ctx)
}

func du(ctx context.Context, values interface{}, args []string) error {

	return nil
}

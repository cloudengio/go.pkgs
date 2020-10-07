package main

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"cloudeng.io/cmdutil"
	"cloudeng.io/cmdutil/subcmd"
)

var (
	cmdSet *subcmd.CommandSet
)

type CommonFlags struct {
	DatabaseDir string `subcmd:"database,$HOME/.filewalk/db,directory containing the file metadata and usage database"`
	Verbose     int    `subcmd:"v,0,higher values show more debugging output"`
}

func init() {
	scanFlagSet := subcmd.NewFlagSet()
	scanFlagSet.MustRegisterFlagStruct(&scanFlags{}, nil, nil)
	summaryFlagSet := subcmd.NewFlagSet()
	summaryFlagSet.MustRegisterFlagStruct(&summaryFlags{}, nil, nil)
	queryFlagSet := subcmd.NewFlagSet()
	queryFlagSet.MustRegisterFlagStruct(&queryFlags{}, nil, nil)
	lsFlagSet := subcmd.NewFlagSet()
	lsFlagSet.MustRegisterFlagStruct(&lsFlags{}, nil, nil)
	eraseFlagSet := subcmd.NewFlagSet()
	eraseFlagSet.MustRegisterFlagStruct(&eraseFlags{}, nil, nil)

	duCmd := subcmd.NewCommand("scan", scanFlagSet, scan)
	duCmd.Document("scan file number and size statistics", "<directory/prefix>+")

	summaryCmd := subcmd.NewCommand("summary", summaryFlagSet, summary, subcmd.WithoutArguments())
	summaryCmd.Document("summarize file count and disk usage")

	queryCmd := subcmd.NewCommand("query", queryFlagSet, query)
	queryCmd.Document("query the file statistics database")

	lsCmd := subcmd.NewCommand("ls", lsFlagSet, ls)
	lsCmd.Document("list the contents of the database")

	eraseCmd := subcmd.NewCommand("erase", eraseFlagSet, erase, subcmd.WithoutArguments())
	eraseCmd.Document("erase the file statistics database")

	cmdSet = subcmd.NewCommandSet(duCmd, eraseCmd, lsCmd, queryCmd, summaryCmd)
}

func main() {
	ctx := context.Background()
	if err := cmdSet.Dispatch(ctx); err != nil {
		cmdutil.Exit("%v", err)
	}
}

type contextKey int

var verbosityKey contextKey = 0

func (c *CommonFlags) withVerbosity(ctx context.Context) context.Context {
	return context.WithValue(ctx, verbosityKey, c.Verbose)
}

func debug(ctx context.Context, level int, format string, args ...interface{}) {
	if level > ctx.Value(verbosityKey).(int) {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	fmt.Printf("%s: %s:% 4d: ", time.Now().Format(time.RFC3339), filepath.Base(file), line)
	fmt.Printf(format, args...)
}

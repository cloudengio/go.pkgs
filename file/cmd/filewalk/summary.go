package main

import (
	"context"
	"fmt"

	"cloudeng.io/file/filewalk/walkdb"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type summaryFlags struct {
	CommonFlags
	TopN int `subcmd:"top,20,show the top prefixes by file count and disk usage"`
}

func printMetric(metric []walkdb.Metric) {
	intPrinter := message.NewPrinter(language.English) // commas in counts.
	for _, m := range metric {
		intPrinter.Printf("%20v : %v\n", m.Size, m.Prefix)
	}
}

func summary(ctx context.Context, values interface{}, args []string) error {
	flagValues := values.(*summaryFlags)
	ctx = flagValues.withVerbosity(ctx)
	db, err := walkdb.Open(flagValues.DatabaseDir, walkdb.ReadOnly())
	if err != nil {
		return err
	}
	nFiles, nChildren, nBytes := db.Totals()
	intPrinter := message.NewPrinter(language.English) // commas in counts.
	intPrinter.Printf("% 20v : total files\n", nFiles)
	intPrinter.Printf("% 20v : total children\n", nChildren)
	intPrinter.Printf("% 20v : total diskUsage\n", nBytes)

	fc := db.FileCounts(flagValues.TopN)
	du := db.DiskUsage(flagValues.TopN)
	ch := db.ChildCounts(flagValues.TopN)
	fmt.Printf("Top %v prefixes by disk usage\n", flagValues.TopN)
	printMetric(du)

	fmt.Printf("Top %v prefixes by file count\n", flagValues.TopN)
	printMetric(fc)

	fmt.Printf("Top %v prefixes by child count\n", flagValues.TopN)
	printMetric(ch)
	return nil
}

package main

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"cloudeng.io/cmdutil"
	"cloudeng.io/errors"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/filewalk/walkdb"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type scanFlags struct {
	CommonFlags
	ConfigFile  string `subcmd:"config,$HOME/.filewalk/config,config file"`
	Describe    bool   `subcmd:"describe-config,false,describe the YAML configuration file"`
	Concurrency int    `subcmd:"concurrency,-1,number of threads to use for scanning"`
	Incremental bool   `subcmd:"incremental,true,incremental mode uses the existing database to avoid as much unnecssary work as possible"`
}

// TODO(cnicolaou): determine a means of adding S3, GCP scanners etc without
// pulling in all of their dependencies into this package and module.
// For example, consider running them as external commands accessing the
// same database (eg. go run cloudeng.io/aws/filewalk ...).

type scanState struct {
	scanner     filewalk.Scanner
	exclusions  Exclusions
	db          *walkdb.Database
	progressCh  chan progressUpdate
	incremental bool
}

type progressUpdate struct {
	prefix int
	files  int
	reused int
}

func (sc *scanState) fileFn(ctx context.Context, prefix string, info *filewalk.Info, ch <-chan filewalk.Contents) error {
	pi := walkdb.PrefixInfo{
		ModTime: info.ModTime,
		Mode:    uint32(info.Mode),
		Size:    info.Size,
	}
	adjuster := sizeFuncFor(prefix)
	debug(ctx, 2, "prefix: %v\n", prefix)
	for results := range ch {
		debug(ctx, 2, "result: %v %v\n", prefix, results.Err)
		if err := results.Err; err != nil {
			if sc.scanner.IsPermissionError(err) {
				debug(ctx, 1, "permission denied: %v\n", prefix)
			} else {
				debug(ctx, 1, "error: %v: %v\n", prefix, err)
			}
			pi.Err = err.Error()
			break
		}
		for _, file := range results.Files {
			pi.DiskUsage += adjuster(file.Size)
			pi.Files = append(pi.Files, walkdb.FileInfo{
				Name:    file.Name,
				Size:    file.Size,
				ModTime: file.ModTime,
			})
		}
		pi.Children = append(pi.Children, results.Children...)
	}
	if err := sc.db.Set(prefix, pi); err != nil {
		return err
	}
	if sc.progressCh != nil {
		sc.progressCh <- progressUpdate{prefix: 1, files: len(pi.Files)}
	}
	return nil
}

func (sc *scanState) prefixFn(ctx context.Context, prefix string, info *filewalk.Info, err error) (bool, []*filewalk.Info, error) {
	if err != nil {
		if sc.scanner.IsPermissionError(err) {
			debug(ctx, 1, "permission denied: %v\n", prefix)
			return true, nil, nil
		}
		debug(ctx, 1, "error: %v\n", prefix)
		return true, nil, err
	}
	if sc.exclusions.Exclude(prefix) {
		debug(ctx, 1, "exclude: %v\n", prefix)
		return true, nil, nil
	}
	if !sc.incremental {
		return false, nil, nil
	}
	prefixInfo, unchanged, err := sc.db.UnchangedDirInfo(prefix, info)
	if err != nil {
		debug(ctx, 1, "error: %v\n", prefix)
		return false, nil, err
	}
	if unchanged {
		if sc.progressCh != nil {
			sc.progressCh <- progressUpdate{reused: len(prefixInfo.Children)}
		}
		debug(ctx, 2, "unchanged: %v: #children: %v\n", prefix, len(prefixInfo.Children))
		// safe to skip unchanged leaf directories.
		return len(prefixInfo.Children) == 0, prefixInfo.Children, nil
	}
	return false, nil, nil
}

func isInteractive() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func scan(ctx context.Context, values interface{}, args []string) error {
	flagValues := values.(*scanFlags)
	ctx = flagValues.withVerbosity(ctx)
	start := time.Now()

	ctx, cancel := context.WithCancel(ctx)
	if flagValues.Describe {
		yamlDocs, err := describeConfigFile()
		fmt.Println(yamlDocs)
		return err
	}
	cfg, err := configFromFile(flagValues.ConfigFile)
	if err != nil {
		return err
	}
	exclusions, err := NewExclusions(cfg.Exclusions...)
	if err != nil {
		return err
	}
	db, err := walkdb.Open(flagValues.DatabaseDir)
	if err != nil {
		return err
	}

	progressCh := make(chan progressUpdate, 100)
	var numPrefixes, numFiles, numReused int64
	sc := scanState{
		exclusions:  exclusions,
		db:          db,
		scanner:     filewalk.LocalScanner(1000),
		progressCh:  progressCh,
		incremental: flagValues.Incremental,
	}

	intPrinter := message.NewPrinter(language.English)
	defer func() {
		intPrinter.Printf("\n")
		intPrinter.Printf("prefixes : % 15v\n", atomic.LoadInt64(&numPrefixes))
		intPrinter.Printf("   files : % 15v\n", atomic.LoadInt64(&numFiles))
		intPrinter.Printf("  reused : % 15v\n", atomic.LoadInt64(&numReused))
		intPrinter.Printf("run time : % 15v\n", time.Since(start))
	}()

	cmdutil.HandleSignals(cancel, os.Interrupt, os.Kill)

	updateDuration := time.Second
	cr := "\r"
	if !isInteractive() {
		updateDuration = time.Second * 30
		cr = "\n"
	}

	go func() {
		last := time.Now()
		for {
			select {
			case update := <-progressCh:
				atomic.AddInt64(&numPrefixes, int64(update.prefix))
				atomic.AddInt64(&numFiles, int64(update.files))
				atomic.AddInt64(&numReused, int64(update.reused))
			case <-ctx.Done():
				return
			}
			if time.Since(last) > updateDuration {
				intPrinter.Printf("prefixes % 15v -- files % 15v -- reused % 15v%s",
					atomic.LoadInt64(&numPrefixes),
					atomic.LoadInt64(&numFiles),
					atomic.LoadInt64(&numReused),
					cr)
				last = time.Now()
			}
		}
	}()

	walker := filewalk.New(sc.scanner, filewalk.Concurrency(flagValues.Concurrency))
	errs := errors.M{}
	errs.Append(walker.Walk(ctx, sc.prefixFn, sc.fileFn, args...))
	errs.Append(db.Persist())
	cancel()
	return errs.Err()
}

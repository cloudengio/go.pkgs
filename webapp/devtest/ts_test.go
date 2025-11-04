package devtest_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cloudeng.io/webapp/devtest"
)

func createFakeTSC(t *testing.T, dir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		script := filepath.Join(dir, "faketsc.bat")
		content := `@echo off
setlocal
echo %*>>tsc_invocations.log
set HAS_TS=0
for %%F in (%*) do (
    echo "%%F" | findstr /E /C:".ts\"" >nul
    if not errorlevel 1 set HAS_TS=1
)
if %HAS_TS%==0 (
    echo no typescript files specified >&2
    exit /b 1
)
:loop
if "%1"=="" goto done
echo %1 | findstr /E ".ts" >nul
if %errorlevel%==0 (
  set "out=%~n1.js"
  echo // compiled %1 > %out%
)
shift
goto loop
:done
`
		if err := os.WriteFile(script, []byte(content), 0700); err != nil { //nolint: gosec //G306
			t.Fatalf("write fake tsc: %v", err)
		}
		return script
	}
	script := filepath.Join(dir, "faketsc.sh")
	content := `#!/bin/sh
echo "$@" >> tsc_invocations.log
# Fail if no arguments or no *.ts files supplied.
HAS_TS=0
for arg in "$@"; do
  case "$arg" in
    *.ts)
      HAS_TS=1
      break
      ;;
  esac
done
if [ $HAS_TS -eq 0 ]; then
  echo "no typescript files specified" >&2
  exit 1
fi
for a in "$@"; do
  case "$a" in
    *.ts)
      out="${a%.ts}.js"
      echo "// compiled $a" > "$out"
      ;;
  esac
done
`
	if err := os.WriteFile(script, []byte(content), 0700); err != nil { //nolint: gosec //G306
		t.Fatalf("write fake tsc: %v", err)
	}
	return script
}

func copyTestdata(t *testing.T, dstDir string, names ...string) {
	t.Helper()
	for _, n := range names {
		src := filepath.Join("testdata", n)
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read testdata %s: %v", src, err)
		}
		if err := os.WriteFile(filepath.Join(dstDir, n), b, 0600); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}
}

func readFile(t *testing.T, fn string) string {
	t.Helper()
	b, err := os.ReadFile(fn)
	if err != nil {
		t.Fatalf("read %s: %v", fn, err)
	}
	return string(b)
}

func TestTypescriptSources_CompileIncremental(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping for windows - needs investigation")
	}
	tmp := t.TempDir()
	copyTestdata(t, tmp, "one.ts", "two.ts")

	fake := createFakeTSC(t, tmp)
	ts := devtest.NewTypescriptSources(
		devtest.WithTypescriptCompiler(fake),
		devtest.WithTypescriptTarget("es2020"),
	)
	ts.SetDirAndFiles(tmp, "one.ts", "two.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ts.Compile(ctx); err != nil {
		t.Fatalf("first compile failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "one.js")); err != nil {
		t.Fatalf("missing one.js: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "two.js")); err != nil {
		t.Fatalf("missing two.js: %v", err)
	}

	log1 := readFile(t, filepath.Join(tmp, "tsc_invocations.log"))
	if !strings.Contains(log1, "one.ts") || !strings.Contains(log1, "two.ts") {
		t.Fatalf("expected both ts files in first invocation, got:\n%s", log1)
	}

	// Record modtime of two.js.
	twoInfoBefore, _ := os.Stat(filepath.Join(tmp, "two.js"))

	time.Sleep(15 * time.Millisecond) // allow mtime granularity

	// Modify only one.ts.
	if err := os.WriteFile(filepath.Join(tmp, "one.ts"), []byte(`export function one(x:number){return x+2}`), 0600); err != nil {
		t.Fatalf("modify one.ts: %v", err)
	}

	if err := ts.Compile(ctx); err != nil {
		t.Fatalf("second compile failed: %v", err)
	}
	log2 := readFile(t, filepath.Join(tmp, "tsc_invocations.log"))
	lines := strings.Split(strings.TrimSpace(log2), "\n")
	last := lines[len(lines)-1]
	if !strings.Contains(last, "one.ts") {
		t.Errorf("expected one.ts in second invocation, got: %s", last)
	}
	if strings.Contains(last, "two.ts") {
		t.Errorf("did not expect two.ts in second invocation, got: %s", last)
	}

	twoInfoAfter, _ := os.Stat(filepath.Join(tmp, "two.js"))
	if !twoInfoAfter.ModTime().Equal(twoInfoBefore.ModTime()) {
		t.Errorf("two.js unexpectedly modified (modtime changed)")
	}
}

func TestTypescriptSources_NoChanges(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping for windows - needs investigation")
	}
	tmp := t.TempDir()
	copyTestdata(t, tmp, "one.ts")

	fake := createFakeTSC(t, tmp)
	ts := devtest.NewTypescriptSources(devtest.WithTypescriptCompiler(fake))
	ts.SetDirAndFiles(tmp, "one.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ts.Compile(ctx); err != nil {
		t.Fatalf("first compile failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := ts.Compile(ctx); err != nil {
		t.Fatalf("second compile (no changes) failed: %v", err)
	}
	logData := readFile(t, filepath.Join(tmp, "tsc_invocations.log"))
	lines := strings.Split(strings.TrimSpace(logData), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one invocations, got:\n%s", logData)
	}
	last := lines[len(lines)-1]
	if !strings.Contains(last, "one.ts") {
		t.Errorf("expected one ts file from first invocation, got: %s", last)
	}
}

func TestTypescriptSources_MissingFileErrorAndCwdRestored(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping for windows - needs investigation")
	}
	tmp := t.TempDir()
	copyTestdata(t, tmp, "one.ts")

	fake := createFakeTSC(t, tmp)
	ts := devtest.NewTypescriptSources(devtest.WithTypescriptCompiler(fake))
	// Include a missing file.
	ts.SetDirAndFiles(tmp, "one.ts", "missing.ts")

	origCWD, _ := os.Getwd()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := ts.Compile(ctx)
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "missing.ts") {
		t.Errorf("error does not reference missing file: %v", err)
	}

	// Check (intended) restoration of working directory. Current implementation
	// does not restore on early error; this test will fail until fixed.
	newCWD, _ := os.Getwd()
	if origCWD != newCWD {
		t.Errorf("working directory not restored after error: got %s want %s", newCWD, origCWD)
	}
}

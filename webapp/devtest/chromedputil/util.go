// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package chromedputil provides utility functions for working with the
// Chrome DevTools Protocol via github.com/chromedp.
package chromedputil

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"text/template"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

//go:embed javascript_functions.js
var javascriptFunctions string

// ListGlobalFunctions returns a list of all global function names defined in the
// current context.
func ListGlobalFunctions(ctx context.Context) ([]string, error) {
	var defined []string
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`Object.getOwnPropertyNames(window).filter(
			key => typeof window[key] === 'function'
		);`, &defined)); err != nil {
		return nil, err
	}
	return defined, nil
}

// WaitForPromise waits for a promise to resolve in the given evaluate parameters.
func WaitForPromise(p *runtime.EvaluateParams) *runtime.EvaluateParams {
	return p.WithAwaitPromise(true)
}

var loadTpl = template.Must(template.New("loadJS").Parse(`
(async () => {
  let r = await chromedp_utils.loadScript("{{.Script}}");
  console.log("Script load result:", r);
  return r;
})();`))

// SourceScript loads a JavaScript script into the current page.
func SourceScript(ctx context.Context, script string) error {
	result := struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}{}
	scr := strings.Builder{}
	data := struct {
		Script string
	}{
		Script: script,
	}
	err := loadTpl.Execute(&scr, data)
	if err != nil {
		return fmt.Errorf("failed to execute load template: %w", err)
	}
	err = chromedp.Run(ctx,
		chromedp.Evaluate(javascriptFunctions, nil),
		chromedp.Evaluate(scr.String(), &result, WaitForPromise),
	)
	if err != nil {
		return fmt.Errorf("failed to evaluate load script %s: %w", script, err)
	}
	if !result.Success {
		return fmt.Errorf("failed to load script: %s", result.Error)
	}
	return nil
}

func evaluate(ctx context.Context, expr string) (*runtime.RemoteObject, error) {
	ro, exp, err := runtime.Evaluate(expr).
		WithSilent(true).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	if exp != nil {
		return nil, fmt.Errorf("javascript exception: %v", exp)
	}
	if ro == nil {
		return nil, fmt.Errorf("failed to get remote object: %s", expr)
	}
	return ro, nil
}

// GetRemoteObjectRef retrieves a remote object's metadata, ie.
// type, object id etc (but not it's value).
func GetRemoteObjectRef(ctx context.Context, name string) (*runtime.RemoteObject, error) {
	var obj *runtime.RemoteObject
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			ro, err := evaluate(ctx, name)
			if err != nil {
				return err
			}
			obj = ro
			return nil
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate %s: %w", name, err)
	}
	return obj, nil
}

func callon(ctx context.Context, fn string, objectID runtime.RemoteObjectID, deep bool) (*runtime.RemoteObject, error) {
	if objectID == "" {
		return nil, fmt.Errorf("missing objectID")
	}
	var res *runtime.RemoteObject
	cp := runtime.CallFunctionOn(fn).
		WithObjectID(objectID)
	if deep {
		cp = cp.WithSerializationOptions(&runtime.SerializationOptions{
			Serialization: runtime.SerializationOptionsSerializationDeep,
		})
	} else {
		cp = cp.WithReturnByValue(true)
	}
	res, exp, err := cp.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to call function %s on object %s: %w", fn, objectID, err)
	}
	if exp != nil {
		return nil, fmt.Errorf("javascript exception: %v (%+v)", exp.Text, exp)
	}
	return res, nil
}

// GetRemoteObjectValueJSON retrieves a remote object's (using ObjectID)
// value using JSON serialization. The object is looked up by its ID and hence
// the supplied ObjectID must be a reference to the object with the ObjectID
// field set. Objects which already contain a JSON value will return that value
// immediately.
// NOTE that GetRemoteObjectValueJSON will return an empty or incomplete serialization
// for platform objects, the ClassName will generally be indicative of whether
// the object is a platform object, e.g Response or Promise.
func GetRemoteObjectValueJSON(ctx context.Context, object *runtime.RemoteObject) (*runtime.RemoteObject, jsontext.Value, error) {
	if object.Value != nil {
		return object, object.Value, nil
	}
	if object.Type == "undefined" {
		object.Value = jsontext.Value(`"undefined"`)
		return object, object.Value, nil
	}
	return safeClone(ctx, object.ObjectID)
}

// IsPlatformObjectError checks if the error is due to a platform object serialization
// error. The only reliable way to determine if an object is a platform object in
// chrome is to attempt a deep serialization and check for this error.
func IsPlatformObjectError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "unknown DeepSerializedValueType value: platformobject")
}

// safeClone uses the javascript safeClone method to attempt to summarize/capture
// platform object details.
func safeClone(ctx context.Context, objectID runtime.RemoteObjectID) (*runtime.RemoteObject, jsontext.Value, error) {
	var obj *runtime.RemoteObject
	err := chromedp.Run(ctx,
		chromedp.Evaluate(javascriptFunctions, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			ro, err := callon(ctx, `function() { return chromedp_utils.safeClone(this); }`, objectID, false)
			if err != nil {
				return err
			}
			obj = ro
			return nil
		}))
	if err != nil {
		return nil, nil, err
	}
	if obj.Type != "" && obj.ClassName != "" {
		return obj, obj.Value, nil
	}
	platformInfo := struct {
		Type      string `json:"_type,omitempty"`
		ClassName string `json:"className,omitempty"`
	}{}
	if err := json.Unmarshal(obj.Value, &platformInfo); err != nil {
		return obj, obj.Value, nil
	}
	if platformInfo.Type != "" {
		obj.ClassName = platformInfo.ClassName
	}
	return obj, obj.Value, nil
}

// IsPlatformObject returns true if the given remote object is a platform object.
// The obj argument must have been obtained via a call to GetRemoteObjectValueJSON.
func IsPlatformObject(obj *runtime.RemoteObject) bool {
	if obj == nil {
		return false
	}
	platformInfo := struct {
		Type string `json:"_type,omitempty"`
	}{}
	if err := json.Unmarshal(obj.Value, &platformInfo); err != nil {
		return false
	}
	return platformInfo.Type != ""
}

// WithExecAllocatorForCI returns a chromedp context with an ExecAllocator
// configured appropriately for CI systems as opposed to when running locally.
// The CI configuration may disable sandboxing for example.
func WithExecAllocatorForCI(ctx context.Context, extraExecAllocOpts ...chromedp.ExecAllocatorOption) (context.Context, func()) {
	chromeBin := ChromeBinPathOnCI()
	modifyCmd := func(cmd *exec.Cmd) {
		fmt.Printf("chrome command line: %v %v\n", cmd.Path, cmd.Args[1:])
	}
	if len(chromeBin) == 0 {
		opts := slices.Clone(chromedp.DefaultExecAllocatorOptions[:])
		opts = append(opts, extraExecAllocOpts...)
		opts = append(opts, chromedp.ModifyCmdFunc(modifyCmd))
		return chromedp.NewExecAllocator(ctx, opts...)
	}
	fmt.Printf("Detected CI environment via CHROME_BIN_PATH=%s\n", chromeBin)
	userDataDir := UserDataDirOnCI()
	fmt.Printf("WARNING: chromedp/chrome: sandboxing disabled\n")
	allOpts := []chromedp.ExecAllocatorOption{
		chromedp.ExecPath(chromeBin),
	}
	if len(userDataDir) > 0 {
		fmt.Printf("Detected CI environment, using  CHROME_USER_DATA_DIR=%s\n", userDataDir)
		allOpts = append(allOpts, chromedp.UserDataDir(userDataDir))
	}
	allOpts = append(allOpts, AllocatorOptsForCI...)
	allOpts = append(allOpts, extraExecAllocOpts...)
	allOpts = append(allOpts, chromedp.ModifyCmdFunc(modifyCmd))
	return chromedp.NewExecAllocator(ctx, allOpts...)
}

// UserDataDirOnCI returns the user data directory for Chrome on CI.
func UserDataDirOnCI() string {
	return os.Getenv("CHROME_USER_DATA_DIR")
}

// ChromeBinPathOnCI returns the Chrome binary path for CI.
func ChromeBinPathOnCI() string {
	return os.Getenv("CHROME_BIN_PATH")
}

var (

	// AllocatorOptsForCI are the default ExecAllocator options for CI environments,
	// they extend chromedp.DefaultExecAllocatorOptions.
	AllocatorOptsForCI = []chromedp.ExecAllocatorOption{
		// Replicating chromedp.DefaultExecAllocatorOptions but
		// with modifications for CI environments and avoiding the possibility
		// of enable/disable features conflicts.

		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", "new"),

		chromedp.Flag("disable-background-networking", true),
		// don't use enable-features + disable-features in the same command line
		// since it's unclear which takes precedence.
		// chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("force-color-profile", "srgb"),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("use-mock-keychain", true),

		// Additional flags for CI.
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-crash-reporter", true),
		chromedp.Flag("disable-component-update", true),
		chromedp.Flag("disable-features", "MetricsReporting,UserMetrics"),
	}
)

// AllocatorOptsVerboseLogging provides ExecAllocator options for verbose logging
// at the specified level.
func AllocatorLoggingWithLevel(level int) []chromedp.ExecAllocatorOption {
	return []chromedp.ExecAllocatorOption{
		chromedp.Flag("enable-logging", "stderr"),
		chromedp.Flag("v", fmt.Sprintf("%d", level)),
	}
}

// WithContextForCI returns a chromedp context that may be different on a CI
// system than when running locally. The CI configuration may disable
// sandboxing etc. The ExecAllocator is always created with appropriate options for
// the various CI environments and extraExecAllocOpts is appended to these.
func WithContextForCI(ctx context.Context, extraExecAllocOpts []chromedp.ExecAllocatorOption, opts ...chromedp.ContextOption) (context.Context, func()) {
	ctx, cancelA := WithExecAllocatorForCI(ctx, extraExecAllocOpts...)
	ctx, cancelB := chromedp.NewContext(ctx, opts...)
	return ctx, func() {
		cancelB()
		cancelA()
	}
}

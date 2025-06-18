// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !darwin

package plugins

import (
	"fmt"
	"io"
	"runtime"
)

// Plugin is a dummy implementation of the Plugin function that returns
// an error on unsupported platforms.
func Plugin(io.Reader, io.Writer) error {
	return fmt.Errorf("this plugin is not supported on %v", runtime.GOOS)
}

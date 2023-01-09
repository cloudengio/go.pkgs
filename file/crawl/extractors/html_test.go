// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package extractors_test

import (
	"context"
	"testing"

	"cloudeng.io/file/crawl/extractors"
)

func TestHTML(t *testing.T) {
	ctx := context.Background()
	errCh := make(chan extractors.Errors, 10)
	he := extractors.NewHTML(errCh)

	errs := []extractors.Errors{}

	go func() {
		for err := range errCh {
			errs = append(errs, err)
		}
	}()

	he.Extract()
}

// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks_test

import (
	"reflect"
	"testing"

	"cloudeng.io/file/crawl/outlinks"
)

func TestOutlinkRegexpProcessor(t *testing.T) {
	proc := outlinks.RegexpProcessor{
		NoFollow: []string{"^http://", "^https://"},
		Follow:   []string{"https://allow.me/"},
		Rewrite:  []string{"s%^(https://allow.me)/(.*?)/(.*)%$1/$3/$2%"},
	}
	if err := proc.Compile(); err != nil {
		t.Fatal(err)
	}

	got := proc.Process([]string{
		"http://www.google.com/",
		"https://www.yahho.com",
		"https://allow.me/one/two/three",
	})
	want := []string{"https://allow.me/two/three/one"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

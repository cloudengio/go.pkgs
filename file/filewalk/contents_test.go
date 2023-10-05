// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk_test

import (
	"os"
	"reflect"
	"testing"

	"cloudeng.io/file/filewalk"
)

func TestEncoding(t *testing.T) {
	el := filewalk.EntryList{
		{Name: "0", Type: os.ModeDir},
		{Name: "1", Type: os.ModeSymlink},
	}
	buf, err := el.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var nel filewalk.EntryList
	if err := nel.UnmarshalBinary(buf); err != nil {
		t.Fatal(err)
	}
	if got, want := nel, el; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", nel, el)
	}
}

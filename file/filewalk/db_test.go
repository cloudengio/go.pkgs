package filewalk_test

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
	"time"

	"cloudeng.io/file/filewalk"
)

func TestCodec(t *testing.T) {
	now := time.Now().Round(0)
	pi := filewalk.PrefixInfo{
		ModTime:    now,
		Size:       33,
		UserID:     "500",
		Mode:       0555,
		DiskUsage:  999,
		DiskLayout: "a string 11",
		Err:        "some err",
	}
	child := filewalk.Info{
		Name:    "file1",
		UserID:  "600",
		Size:    3444,
		ModTime: now,
		Mode:    0666,
	}
	pi.Files = []filewalk.Info{child, child}
	pi.Children = []filewalk.Info{child, child}
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(pi); err != nil {
		t.Fatal(err)
	}

	dec := gob.NewDecoder(bytes.NewBuffer(buf.Bytes()))
	var npi filewalk.PrefixInfo
	if err := dec.Decode(&npi); err != nil {
		t.Fatal(err)
	}
	if got, want := pi, npi; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

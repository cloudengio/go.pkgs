// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content_test

import (
	"reflect"
	"strings"
	"testing"

	"cloudeng.io/file/content"
)

type handlerType struct {
	name string
}

func TestRegistryConverter(t *testing.T) {
	reg := content.NewRegistry[*handlerType]()
	var err error

	register := func(from, to string, handler *handlerType) {
		if err = reg.RegisterConverters(content.Type(from), content.Type(to), handler); err != nil {
			t.Fatal(err)
		}
	}

	lookup := func(from, to string, handler *handlerType) {
		got, err := reg.LookupConverters(content.Type(from), content.Type(to))
		if err != nil {
			t.Error(err)
			return
		}
		if want := handler; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := got.name, handler.name; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	handler := &handlerType{name: "my/type"}
	register("text/html", "text/plain", handler)
	register("text/html;charset=utf-8", "text/plain", handler)
	lookup("text/html", "text/plain", handler)
	lookup("text/html;charset=utf-8", "text/plain", handler)
	lookup("text/html ;charset=utf-8", "text/plain", handler)
	lookup("text/html; charset=utf-8", "text/plain", handler)
	lookup("text/html ; charset=utf-8", "text/plain", handler)

	_, err = reg.LookupConverters("text/html", "text/plainx")
	if err == nil || !strings.Contains(err.Error(), "no converter for") {
		t.Fatal(err)
	}
	err = reg.RegisterConverters("text/html", "text/plain", handler)
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatal(err)
	}
}

func TestRegistryHandler(t *testing.T) {
	reg := content.NewRegistry[*handlerType]()
	var err error

	handler := []*handlerType{{name: "my/type"}}

	if err = reg.RegisterHandlers("my/type", handler...); err != nil {
		t.Fatal(err)
	}

	got, err := reg.LookupHandlers("my/type")
	if err != nil {
		t.Error(err)
	}

	if got, want := got[0].name, handler[0].name; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if want := handler; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

}

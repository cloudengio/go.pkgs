package flags_test

import (
	"errors"
	"flag"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/flags"
)

func ExampleRangeSpecs() {
	irs := flags.IntRangeSpecs{}
	if err := irs.Set("1-2,3-4,-5,8-"); err != nil {
		panic(err)
	}
	fmt.Println(irs)
	// Output:
	// [{1 2 false false} {3 4 false false} {5 0 true false} {8 0 false true}]
}

func newRangeSpec(f, t string, rel, ext bool) flags.RangeSpec {
	return flags.RangeSpec{
		From:          f,
		To:            t,
		RelativeToEnd: rel,
		ExtendsToEnd:  ext,
	}
}

func newIntRangeSpec(f, t int, rel, ext bool) flags.IntRangeSpec {
	return flags.IntRangeSpec{
		From:          f,
		To:            t,
		RelativeToEnd: rel,
		ExtendsToEnd:  ext,
	}
}

func newColonRangeSpec(f, t string, rel, ext bool) flags.ColonRangeSpec {
	return flags.ColonRangeSpec{newRangeSpec(f, t, rel, ext)}
}

func TestStringRange(t *testing.T) {
	spec := newRangeSpec
	cspec := newColonRangeSpec

	for i, tc := range []struct {
		input    string
		from, to string
		rel, ext bool
	}{
		{"a", "a", "", false, false},
		{"a-", "a", "", false, true},
		{"-a", "a", "", true, false},
		{"-a-", "a", "", true, true},
		{"ᚠᛇᚻ", "ᚠᛇᚻ", "", false, false},
		{"ᚠᛇᚻ-", "ᚠᛇᚻ", "", false, true},
		{"-ᚠᛇᚻ", "ᚠᛇᚻ", "", true, false},
		{"-ᚠᛇᚻ-", "ᚠᛇᚻ", "", true, true},
		{"a-b", "a", "b", false, false},
		{"-a-b", "a", "b", true, false},
	} {
		var sr flags.RangeSpec
		if err := sr.Set(tc.input); err != nil {
			t.Errorf("%v: %v: %v", i, tc.input, err)
			continue
		}
		if got, want := sr, spec(tc.from, tc.to, tc.rel, tc.ext); !reflect.DeepEqual(got, want) {
			t.Errorf("%v: %v: got %v, want %v", i, tc.input, got, want)
		}
		if got, want := sr.String(), tc.input; got != want {
			t.Errorf("%v: %v: got %v, want %v", i, tc.input, got, want)
		}

		var csr flags.ColonRangeSpec
		tc.input = strings.ReplaceAll(tc.input, "-", ":")
		if err := csr.Set(tc.input); err != nil {
			t.Errorf("%v: %v: %v", i, tc.input, err)
			continue
		}
		if got, want := csr, cspec(tc.from, tc.to, tc.rel, tc.ext); !reflect.DeepEqual(got, want) {
			t.Errorf("%v: %v: got %v, want %v", i, tc.input, got, want)
		}
		if got, want := csr.String(), tc.input; got != want {
			t.Errorf("%v: %v: got %v, want %v", i, tc.input, got, want)
		}

	}

	for _, tc := range []string{
		"", "-", "--", "---", "a,b", "a-b-c", "ab--cc", "ab--",
	} {
		var sr flags.RangeSpec
		err := sr.Set(tc)
		if err == nil {
			t.Errorf("%q: expected an error but got none", tc)
			continue
		}
		if !errors.Is(err, &flags.ErrInvalidRange{}) {
			t.Errorf("%v: error is of the wrong kind: %T", tc, err)
		}
	}

	for _, tc := range []string{
		"", ":", "::", ":::", "a,b", "a:b:c", "ab::cc", "ab::",
	} {
		var sr flags.ColonRangeSpec
		err := sr.Set(tc)
		if err == nil {
			t.Errorf("%q: expected an error but got none", tc)
			continue
		}
		if !errors.Is(err, &flags.ErrInvalidRange{}) {
			t.Errorf("%v: error is of the wrong kind: %T", tc, err)
		}
	}
}

func TestStringRanges(t *testing.T) {
	spec := newRangeSpec
	cspec := newColonRangeSpec
	rs := flags.RangeSpecs{}
	if err := rs.Set("1-2,-4-5,a-"); err != nil {
		t.Fatal(err)
	}
	if got, want := len(rs), 3; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got, want := rs[1], spec("4", "5", true, false); !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}

	if got, want := rs[2], spec("a", "", false, true); !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}

	if got, want := rs.String(), "1-2,-4-5,a-"; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	crs := flags.ColonRangeSpecs{}
	if err := crs.Set("1:2,:4:5,a:"); err != nil {
		t.Fatal(err)
	}
	if got, want := len(crs), 3; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got, want := rs[1], spec("4", "5", true, false); !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
	if got, want := crs[2], cspec("a", "", false, true); !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
	if got, want := crs.String(), "1:2,:4:5,a:"; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}

	err := rs.Set("1-2,-4--,a-")
	if err == nil || !errors.Is(err, &flags.ErrInvalidRange{}) {
		t.Errorf("missing or wrong error: %v", err)
	}
	err = crs.Set("1:2,:4::,a:")
	if err == nil || !errors.Is(err, &flags.ErrInvalidRange{}) {
		t.Errorf("missing or wrong error: %v", err)
	}
}

func TestIntRanges(t *testing.T) {
	ispec := newIntRangeSpec
	irs := flags.IntRangeSpecs{}
	if err := irs.Set("1-2,3-4,-5,8-"); err != nil {
		t.Fatal(err)
	}

	if got, want := []flags.IntRangeSpec(irs), []flags.IntRangeSpec{
		ispec(1, 2, false, false),
		ispec(3, 4, false, false),
		ispec(5, 0, true, false),
		ispec(8, 0, false, true),
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := irs.String(), "1-2,3-4,-5,8-"; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	err := irs.Set("a-1")
	if err == nil || !errors.Is(err, &flags.ErrInvalidRange{}) {
		t.Errorf("missing or wrong error: %v", err)
	}
	err = irs.Set("1-a")
	if err == nil || !errors.Is(err, &flags.ErrInvalidRange{}) {
		t.Errorf("missing or wrong error: %v", err)
	}
	t.Log(err)
	ir := flags.IntRangeSpec{}
	err = ir.Set("---a")
	if err == nil || !errors.Is(err, &flags.ErrInvalidRange{}) {
		t.Errorf("missing or wrong error: %v", err)
	}
	t.Log(err)
}

func TestRangeFlags(t *testing.T) {
	rangeFlags := struct {
		A flags.RangeSpecs    `cmd:"strings,,string ranges"`
		B flags.IntRangeSpecs `cmd:"ints,,int ranges"`
	}{}
	flagSet := &flag.FlagSet{}
	err := flags.RegisterFlagsInStruct(flagSet, "cmd", &rangeFlags, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	flagSet.Parse([]string{"--strings=a:b,c:d", "--ints=-3,6-7"})
	if got, want := rangeFlags.A.String(), "a:b,c:d"; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got, want := rangeFlags.B.String(), "-3,6-7"; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

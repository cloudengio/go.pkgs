// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package zipcode_test

import (
	"reflect"
	"testing"

	"cloudeng.io/geospatial/zipcode"
)

const sampleData = `
US	99553	Akutan	Alaska	AK	Aleutians East	013			54.143	-165.7854	1
GB	BN91	Worthing	England	ENG					50.818	-0.3754	
GB	AL3 8QE	Slip End	England	ENG	Bedfordshire		Central Bedfordshire	E06000056	51.8479	-0.4474	6
`

func TestLatLong(t *testing.T) {
	// Test the latitude and longitude of a zipcode.
	zdb := zipcode.NewDB()
	if err := zdb.Load([]byte(sampleData)); err != nil {
		t.Fatalf("failed to load sample data: %v", err)
	}
	ll, _ := zdb.LatLong("AK", "99553")
	if got, want := ll, (zipcode.LatLong{54.143, -165.7854}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	ll, _ = zdb.LatLong("ENG", "BN91")
	if got, want := ll, (zipcode.LatLong{50.818, -0.3754}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	ll, _ = zdb.LatLong("ENG", "AL3 8QE")
	if got, want := ll, (zipcode.LatLong{51.8479, -0.4474}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	_, ok := zdb.LatLong("ENG", "AL3 8QF")
	if ok {
		t.Errorf("expected not to find AL3 8QF")
	}
}

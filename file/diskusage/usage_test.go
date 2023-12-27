// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package diskusage_test

import (
	"fmt"
	"testing"

	"cloudeng.io/file/diskusage"
)

func ExampleBase2Bytes() {
	fmt.Println(diskusage.KiB.Num(512))
	fmt.Println(diskusage.KiB.Num(2048))
	fmt.Println(diskusage.GiB.Num(1073741824))
	fmt.Println(diskusage.Base2Bytes(1024).Standardize())
	fmt.Println(diskusage.Base2Bytes(1536).Standardize())
	fmt.Println(diskusage.Base2Bytes(1610612736).Standardize())
	// Output:
	// 0.5
	// 2
	// 1
	// 1 KiB
	// 1.5 KiB
	// 1.5 GiB
}

func ExampleDecimalBytes() {
	fmt.Println(diskusage.KB.Num(500))
	fmt.Println(diskusage.KB.Num(2000))
	fmt.Println(diskusage.GB.Num(1000000000))
	fmt.Println(diskusage.DecimalBytes(1000).Standardize())
	fmt.Println(diskusage.DecimalBytes(1500).Standardize())
	fmt.Println(diskusage.DecimalBytes(1500000000).Standardize())
	// Output:
	// 0.5
	// 2
	// 1
	// 1 KB
	// 1.5 KB
	// 1.5 GB
}

func TestBytesParser(t *testing.T) {
	for _, tc := range []struct {
		input    string
		expected float64
	}{
		{"1.1EB", 1.1 * float64(diskusage.EB)},
		{"1.1PB", 1.1 * float64(diskusage.PB)},
		{"1.1PB", 1.1 * float64(diskusage.PB)},
		{"1.1GB", 1.1 * float64(diskusage.GB)},
		{"1.1MB", 1.1 * float64(diskusage.MB)},
		{"1.1KB", 1.1 * float64(diskusage.KB)},
		{"1.1EiB", 1.1 * float64(diskusage.EiB)},
		{"1.1PiB", 1.1 * float64(diskusage.PiB)},
		{"1.1PiB", 1.1 * float64(diskusage.PiB)},
		{"1.1GiB", 1.1 * float64(diskusage.GiB)},
		{"1.1MiB", 1.1 * float64(diskusage.MiB)},
		{"1.1KiB", 1.1 * float64(diskusage.KiB)},
		{"1000", 1000},
		{"100,000", 100000},
	} {
		out, err := diskusage.ParseToBytes(tc.input)
		if err != nil {
			t.Errorf("ParseToBytes(%v): %v", tc.input, err)
		}
		if got, want := out, tc.expected; got != want {
			t.Errorf("ParseToBytes(%v): got %v, want %v", tc.input, got, want)
		}
	}
}

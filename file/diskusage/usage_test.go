// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package diskusage_test

import (
	"fmt"

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
	// 1 KiB
	// 1.5 B
	// 1.5 GB
}

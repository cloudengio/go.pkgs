// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package diskusage_test

import (
	"fmt"
	"testing"

	"cloudeng.io/file/diskusage"
)

func ExampleBinary() {
	fmt.Println(diskusage.KiB.Value(512))
	fmt.Println(diskusage.KiB.Value(2048))
	fmt.Println(diskusage.GiB.Value(1073741824))
	fmt.Println(diskusage.Binary(1024).Standardize())
	fmt.Println(diskusage.Binary(1536).Standardize())
	fmt.Println(diskusage.Binary(1610612736).Standardize())
	fmt.Printf("%.0f\n", diskusage.Binary(512)) // Default precision
	// Output:
	// 0.5
	// 2
	// 1
	// 1 KiB
	// 1.5 KiB
	// 1.5 GiB
	// 512 B
}

func ExampleDecimal() {
	fmt.Println(diskusage.KB.Value(500))
	fmt.Println(diskusage.KB.Value(2000))
	fmt.Println(diskusage.GB.Value(1000000000))
	fmt.Println(diskusage.Decimal(1000).Standardize())
	fmt.Println(diskusage.Decimal(1500).Standardize())
	fmt.Println(diskusage.Decimal(1500000000).Standardize())
	fmt.Printf("%.0f\n", diskusage.Decimal(500)) // Default precision
	// Output:
	// 0.5
	// 2
	// 1
	// 1 KB
	// 1.5 KB
	// 1.5 GB
	// 500 B
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

// GenAI: gemini 2.5 wrote this code. Needed several fixes though.
func TestFormatters(t *testing.T) {
	tests := []struct {
		name       string
		value      diskusage.SizeUnit
		isBinary   bool
		formatVerb string
		expected   string
	}{
		// Decimal Tests
		{
			name:       "Decimal Bytes default",
			value:      diskusage.SizeUnit(500),
			isBinary:   false,
			formatVerb: "%f", // Default precision 2
			expected:   "500.00 B",
		},
		{
			name:       "Decimal Bytes specific precision",
			value:      diskusage.SizeUnit(500),
			isBinary:   false,
			formatVerb: "%.1f",
			expected:   "500.0 B",
		},
		{
			name:       "Decimal KB default",
			value:      diskusage.SizeUnit(1500), // 1.5 KB
			isBinary:   false,
			formatVerb: "%f",
			expected:   "1.50 KB",
		},
		{
			name:       "Decimal KB specific precision",
			value:      diskusage.SizeUnit(1500),
			isBinary:   false,
			formatVerb: "%.1f",
			expected:   "1.5 KB",
		},
		{
			name:       "Decimal KB with width",
			value:      diskusage.SizeUnit(1500),
			isBinary:   false,
			formatVerb: "%10.1f",
			expected:   "       1.5 KB",
		},
		{
			name:       "Decimal MB default",
			value:      diskusage.MB * 2,
			isBinary:   false,
			formatVerb: "%f",
			expected:   "2.00 MB",
		},
		{
			name:       "Decimal Bytes %g (prec 2)",
			value:      diskusage.SizeUnit(500),
			isBinary:   false,
			formatVerb: "%.2g",
			expected:   "5e+02 B", // 500.0 with %.2g
		},
		{
			name:       "Decimal KB %g (prec 2)",
			value:      diskusage.SizeUnit(1500), // 1.5 KB
			isBinary:   false,
			formatVerb: "%g",
			expected:   "1.5 KB", // 1.5 with %.2g
		},
		{
			name:       "Decimal MB %g (prec 2)",
			value:      diskusage.MB * 2, // 2.0 MB
			isBinary:   false,
			formatVerb: "%g",
			expected:   "2 MB", // 2.0 with %.2g
		},
		{
			name:       "Decimal KB %.0g (integer)",
			value:      diskusage.SizeUnit(1500), // 1.5 KB, int(1.5) = 1
			isBinary:   false,
			formatVerb: "%.0g",
			expected:   "1 KB",
		},
		{
			name:       "Decimal MB %.0g (integer)",
			value:      diskusage.MB * 2, // 2.0 MB, int(2.0) = 2
			isBinary:   false,
			formatVerb: "%.0g",
			expected:   "2 MB",
		},
		{
			name:       "Decimal Bytes %v (default verb)",
			value:      diskusage.SizeUnit(500),
			isBinary:   false,
			formatVerb: "%v",
			expected:   "500.00 B", // Should behave like %f
		},
		{
			name:       "Decimal Zero %f",
			value:      diskusage.SizeUnit(0),
			isBinary:   false,
			formatVerb: "%f",
			expected:   "0.00 B",
		},
		{
			name:       "Decimal Zero %g",
			value:      diskusage.SizeUnit(0),
			isBinary:   false,
			formatVerb: "%g",
			expected:   "0 B",
		},
		{
			name:       "Decimal Zero %.0g",
			value:      diskusage.SizeUnit(0),
			isBinary:   false,
			formatVerb: "%.0g",
			expected:   "0 B",
		},
		{
			name:       "Decimal Negative %f",
			value:      diskusage.SizeUnit(-100),
			isBinary:   false,
			formatVerb: "%f",
			expected:   "-100.00 B",
		},

		// Binary Tests
		{
			name:       "Binary Bytes default",
			value:      diskusage.SizeUnit(500),
			isBinary:   true,
			formatVerb: "%f", // Default precision 2
			expected:   "500.00 B",
		},
		{
			name:       "Binary KiB default",
			value:      diskusage.SizeUnit(1536), // 1.5 KiB (1024 * 1.5)
			isBinary:   true,
			formatVerb: "%f",
			expected:   "1.50 KiB",
		},
		{
			name:       "Binary KiB specific precision",
			value:      diskusage.SizeUnit(1536),
			isBinary:   true,
			formatVerb: "%.1f",
			expected:   "1.5 KiB",
		},
		{
			name:       "Binary MiB default",
			value:      diskusage.MiB * 2,
			isBinary:   true,
			formatVerb: "%f",
			expected:   "2.00 MiB",
		},
		{
			name:       "Binary Bytes %g (prec 2)",
			value:      diskusage.SizeUnit(500),
			isBinary:   true,
			formatVerb: "%g",
			expected:   "5e+02 B", // 500.0 with %.2g
		},
		{
			name:       "Binary KiB %g (prec 2)",
			value:      diskusage.SizeUnit(1536), // 1.5 KiB
			isBinary:   true,
			formatVerb: "%g",
			expected:   "1.5 KiB", // 1.5 with %.2g
		},
		{
			name:       "Binary MiB %g (prec 2)",
			value:      diskusage.MiB * 2, // 2.0 MiB
			isBinary:   true,
			formatVerb: "%g",
			expected:   "2 MiB", // 2.0 with %.2g
		},
		{
			name:       "Binary KiB %.0g (integer)",
			value:      diskusage.SizeUnit(1536), // 1.5 KiB, int(1.5) = 1
			isBinary:   true,
			formatVerb: "%.0g",
			expected:   "1 KiB",
		},
		{
			name:       "Binary MiB %.0g (integer)",
			value:      diskusage.MiB * 2, // 2.0 MiB, int(2.0) = 2
			isBinary:   true,
			formatVerb: "%.0g",
			expected:   "2 MiB",
		},
		{
			name:       "Binary Bytes %v (default verb)",
			value:      diskusage.SizeUnit(500),
			isBinary:   true,
			formatVerb: "%v",
			expected:   "500.00 B", // Should behave like %f
		},
		{
			name:       "Binary Zero %f",
			value:      diskusage.SizeUnit(0),
			isBinary:   true,
			formatVerb: "%f",
			expected:   "0.00 B",
		},
		{
			name:       "Binary Negative %f",
			value:      diskusage.SizeUnit(-100),
			isBinary:   true,
			formatVerb: "%f",
			expected:   "-100.00 B",
		},
		{
			name:       "Decimal EB",
			value:      diskusage.EB + diskusage.PB*200, // 1.2 EB
			isBinary:   false,
			formatVerb: "%.2f",
			expected:   "1.20 EB",
		},
		{
			name:       "Binary EiB",
			value:      diskusage.EiB + diskusage.PiB*198, // 1.19... EiB
			isBinary:   true,
			formatVerb: "%.2f",
			expected:   "1.19 EiB", // (1 + 200.0/1024.0) EiB = 1.1953125 EiB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.isBinary {
				result = fmt.Sprintf(tt.formatVerb, diskusage.Binary(tt.value))
			} else {
				result = fmt.Sprintf(tt.formatVerb, diskusage.Decimal(tt.value))
			}
			if result != tt.expected {
				t.Errorf("%v: Format(%s, %v) = %q, want %q", tt.name, tt.formatVerb, tt.value, result, tt.expected)
			}
		})
	}
}

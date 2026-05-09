// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dsql

import "testing"

func TestClusterIDFromIdentifier(t *testing.T) {
	const bareID = "abcdefghij0123456789abcdef" // 26 chars

	for _, tc := range []struct {
		input   string
		want    string
		wantErr bool
	}{
		{bareID, bareID, false},
		{bareID + ".dsql.us-east-1.on.aws", bareID, false},
		{bareID + ".dsql.eu-west-1.amazonaws.com", bareID, false},
		{"", "", true},
		{"tooshort", "", true},
		{"ABCDEFGHIJ0123456789ABCDEF", "", true}, // uppercase not allowed
		{"abc def ghij0123456789abcd", "", true}, // spaces
		{"has-hyphens-0123456789abcd", "", true}, // hyphens
	} {
		got, err := clusterIDFromIdentifier(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("clusterIDFromIdentifier(%q): expected error, got %q", tc.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("clusterIDFromIdentifier(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("clusterIDFromIdentifier(%q): got %q, want %q", tc.input, got, tc.want)
		}
	}
}

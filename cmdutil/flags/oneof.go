// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"fmt"
	"sort"
	"strings"
)

// OneOf represents a string that can take only one of a fixed set of
// values.
type OneOf string

// Validate ensures that the instance of OneOf has one of the specified set
// values.
func (ef OneOf) Validate(value string, values ...string) error {
	allowed := append(values, value)
	for _, val := range allowed {
		if string(ef) == val {
			return nil
		}
	}
	sort.Strings(allowed)
	return fmt.Errorf("unrecognised flag value: %q is not one of: %s", ef, strings.Join(allowed, ", "))
}

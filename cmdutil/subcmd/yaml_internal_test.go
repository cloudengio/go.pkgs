// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd

import (
	"fmt"
	"os"
	"strings"
)

func (c *CommandSetYAML) Usage(names ...string) string {
	pathname := strings.Join(names, "/")
	if cmd := c.cmdDict[pathname]; cmd != nil {
		return cmd.Usage()
	}
	fmt.Printf("d: %v\n", c.cmdDict)
	return c.CommandSet.Usage(os.Args[0])
}

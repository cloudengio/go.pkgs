// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"context"
	"fmt"
	"os/exec"
)

func GoBuild(ctx context.Context, binary string, args ...string) (string, error) {
	binary = ExecName(binary)
	cmd := exec.CommandContext(ctx, "go", append([]string{"build", "-o", binary}, args...)...) // #nosec G204
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %v", out, err)
	}
	return binary, nil
}

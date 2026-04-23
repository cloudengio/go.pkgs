// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import "os"

// IsGitHubActions returns true when running inside any GitHub Actions workflow,
// regardless of whether the runner is hosted or self-hosted.
func IsGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// IsGitHubHostedRunner returns true when running on a GitHub-hosted runner.
func IsGitHubHostedRunner() bool {
	return IsGitHubActions() && os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted"
}

// IsSelfHostedRunner returns true when running on a self-hosted GitHub Actions runner.
func IsSelfHostedRunner() bool {
	return IsGitHubActions() && os.Getenv("RUNNER_ENVIRONMENT") == "self-hosted"
}

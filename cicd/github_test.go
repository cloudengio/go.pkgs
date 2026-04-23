// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import "testing"

func TestIsGitHubActions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	if IsGitHubActions() {
		t.Error("expected false when GITHUB_ACTIONS is unset")
	}
	t.Setenv("GITHUB_ACTIONS", "true")
	if !IsGitHubActions() {
		t.Error("expected true when GITHUB_ACTIONS=true")
	}
	t.Setenv("GITHUB_ACTIONS", "1") // only "true" counts
	if IsGitHubActions() {
		t.Error("expected false when GITHUB_ACTIONS=1")
	}
}

func TestIsGitHubHostedRunner(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("RUNNER_ENVIRONMENT", "github-hosted")
	if !IsGitHubHostedRunner() {
		t.Error("expected true for github-hosted runner")
	}
	if IsSelfHostedRunner() {
		t.Error("expected false for self-hosted when environment is github-hosted")
	}

	t.Setenv("RUNNER_ENVIRONMENT", "self-hosted")
	if IsGitHubHostedRunner() {
		t.Error("expected false for github-hosted when environment is self-hosted")
	}
	if !IsSelfHostedRunner() {
		t.Error("expected true for self-hosted runner")
	}

	t.Setenv("GITHUB_ACTIONS", "")
	if IsGitHubHostedRunner() || IsSelfHostedRunner() {
		t.Error("expected both false when not on GitHub Actions")
	}
}

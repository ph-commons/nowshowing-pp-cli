// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsBaseURLHostAllowed covers the exact-match host-allowlist logic in
// isolation: no substring/suffix bypass, www variant allowed, trailing-dot
// FQDN normalized, empty input short-circuits to allowed, and malformed
// input surfaces a parse error rather than silently allowing or denying.
// Issue #13 (M3).
func TestIsBaseURLHostAllowed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		rawURL      string
		wantAllowed bool
		wantErr     bool
	}{
		{"default host", "https://www.clickthecity.com", true, false},
		{"bare host allowed", "https://clickthecity.com", true, false},
		{"www host allowed", "https://www.clickthecity.com/some/path", true, false},
		{"bare trailing dot FQDN allowed", "https://clickthecity.com./now-playing", true, false},
		{"www trailing dot FQDN allowed", "https://www.clickthecity.com.", true, false},
		{"empty is allowed (unconfigured)", "", true, false},

		// No substring/suffix bypass.
		{"prefix lookalike denied", "https://notclickthecity.com", false, false},
		{"suffix lookalike denied", "https://clickthecity.com.evil.com", false, false},
		{"concatenated lookalike denied", "https://evilclickthecity.com", false, false},
		{"userinfo trick denied", "https://clickthecity.com@evil.com/", false, false},
		{"unrelated host denied", "http://internal.example.com", false, false},
		{"ip literal denied", "http://169.254.169.254/latest/meta-data/", false, false},

		// Malformed input fails closed (parse error, not a silent allow).
		{"embedded whitespace fails to parse", "https://clickthecity.com  ", false, true},
		{"bad percent escape fails to parse", "https://clickthecity.com%2eevil.com", false, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			allowed, _, err := isBaseURLHostAllowed(tc.rawURL)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("isBaseURLHostAllowed(%q) err = nil, want non-nil", tc.rawURL)
				}
				if allowed {
					t.Fatalf("isBaseURLHostAllowed(%q) allowed = true on parse error, want false (fail closed)", tc.rawURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("isBaseURLHostAllowed(%q) unexpected err: %v", tc.rawURL, err)
			}
			if allowed != tc.wantAllowed {
				t.Fatalf("isBaseURLHostAllowed(%q) allowed = %v, want %v", tc.rawURL, allowed, tc.wantAllowed)
			}
		})
	}
}

// nonExistentConfigPath returns a path guaranteed not to exist, so Load
// falls back to its BaseURL default/env behavior without picking up any
// real on-disk config.toml from the test machine.
func nonExistentConfigPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "does-not-exist-config.toml")
}

// TestLoadDefaultBaseURLWorks is the acceptance-criteria test: "Default CTC
// URL still works" — Load with no overrides must succeed and keep the
// default clickthecity.com BaseURL.
func TestLoadDefaultBaseURLWorks(t *testing.T) {
	t.Setenv("NOWSHOWING_BASE_URL", "")
	t.Setenv(allowCustomBaseURLEnvVar, "0")

	cfg, err := Load(nonExistentConfigPath(t), false)
	if err != nil {
		t.Fatalf("Load with no overrides returned error: %v", err)
	}
	if cfg.BaseURL != "https://www.clickthecity.com" {
		t.Fatalf("cfg.BaseURL = %q, want default https://www.clickthecity.com", cfg.BaseURL)
	}
}

// TestLoadDeniesNonAllowlistedHostWithoutBreakGlass is the acceptance-
// criteria test: "BaseURL to non-allowlisted host fails closed without
// break-glass."
func TestLoadDeniesNonAllowlistedHostWithoutBreakGlass(t *testing.T) {
	t.Setenv("NOWSHOWING_BASE_URL", "http://evil.example.com")
	t.Setenv(allowCustomBaseURLEnvVar, "0")

	cfg, err := Load(nonExistentConfigPath(t), false)
	if err == nil {
		t.Fatalf("Load with disallowed host and no break-glass returned nil error; cfg = %+v", cfg)
	}
	if cfg != nil {
		t.Fatalf("Load returned non-nil *Config on deny path: %+v", cfg)
	}
	if !strings.Contains(err.Error(), "evil.example.com") {
		t.Fatalf("Load error %q does not name the rejected host", err.Error())
	}
}

// TestLoadDeniesMalformedBaseURL covers the parse-error path found during
// plan red-team review: a NOWSHOWING_BASE_URL that fails to parse must fail
// closed exactly like a resolved-but-disallowed host, never silently pass
// through.
func TestLoadDeniesMalformedBaseURL(t *testing.T) {
	t.Setenv("NOWSHOWING_BASE_URL", "https://clickthecity.com  ")
	t.Setenv(allowCustomBaseURLEnvVar, "0")

	cfg, err := Load(nonExistentConfigPath(t), false)
	if err == nil {
		t.Fatalf("Load with malformed BaseURL returned nil error; cfg = %+v", cfg)
	}
	if cfg != nil {
		t.Fatalf("Load returned non-nil *Config on malformed-BaseURL deny path: %+v", cfg)
	}
}

// TestLoadAllowsNonAllowlistedHostViaFlag covers the --allow-custom-base-url
// break-glass path (the allowCustomBaseURL parameter).
func TestLoadAllowsNonAllowlistedHostViaFlag(t *testing.T) {
	t.Setenv("NOWSHOWING_BASE_URL", "http://staging.internal.example.com")
	t.Setenv(allowCustomBaseURLEnvVar, "0")

	cfg, err := Load(nonExistentConfigPath(t), true)
	if err != nil {
		t.Fatalf("Load with disallowed host and allowCustomBaseURL=true returned error: %v", err)
	}
	if cfg.BaseURL != "http://staging.internal.example.com" {
		t.Fatalf("cfg.BaseURL = %q, want http://staging.internal.example.com", cfg.BaseURL)
	}
}

// TestLoadAllowsNonAllowlistedHostViaEnvBreakGlass covers the
// NOWSHOWING_ALLOW_CUSTOM_BASE_URL=1 break-glass path — the mechanism
// callers with no CLI flag surface (e.g. the MCP server) must rely on.
func TestLoadAllowsNonAllowlistedHostViaEnvBreakGlass(t *testing.T) {
	t.Setenv("NOWSHOWING_BASE_URL", "http://staging.internal.example.com")
	t.Setenv(allowCustomBaseURLEnvVar, "1")

	cfg, err := Load(nonExistentConfigPath(t), false)
	if err != nil {
		t.Fatalf("Load with disallowed host and env break-glass returned error: %v", err)
	}
	if cfg.BaseURL != "http://staging.internal.example.com" {
		t.Fatalf("cfg.BaseURL = %q, want http://staging.internal.example.com", cfg.BaseURL)
	}
}

// TestLoadDeniesNonAllowlistedConfigFileBaseURL covers the "config BaseURL
// host" half of the issue title: the allowlist must gate a value that came
// from the TOML config file, not just the NOWSHOWING_BASE_URL env var.
func TestLoadDeniesNonAllowlistedConfigFileBaseURL(t *testing.T) {
	t.Setenv("NOWSHOWING_BASE_URL", "")
	t.Setenv(allowCustomBaseURLEnvVar, "0")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`base_url = "http://evil.example.com"`+"\n"), 0o600); err != nil {
		t.Fatalf("writing temp config file: %v", err)
	}

	cfg, err := Load(path, false)
	if err == nil {
		t.Fatalf("Load with disallowed config-file base_url returned nil error; cfg = %+v", cfg)
	}
	if !strings.Contains(err.Error(), "evil.example.com") {
		t.Fatalf("Load error %q does not name the rejected host", err.Error())
	}
}

// TestLoadAllowsAllowlistedConfigFileBaseURL is the config-file-side allow
// counterpart: a config file explicitly setting base_url to an allowlisted
// host must still load successfully.
func TestLoadAllowsAllowlistedConfigFileBaseURL(t *testing.T) {
	t.Setenv("NOWSHOWING_BASE_URL", "")
	t.Setenv(allowCustomBaseURLEnvVar, "0")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`base_url = "https://clickthecity.com"`+"\n"), 0o600); err != nil {
		t.Fatalf("writing temp config file: %v", err)
	}

	cfg, err := Load(path, false)
	if err != nil {
		t.Fatalf("Load with allowlisted config-file base_url returned error: %v", err)
	}
	if cfg.BaseURL != "https://clickthecity.com" {
		t.Fatalf("cfg.BaseURL = %q, want https://clickthecity.com", cfg.BaseURL)
	}
}

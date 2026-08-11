// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ph-commons/nowshowing-pp-cli/internal/cliutil"
)

func TestGetBytesOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != DefaultUserAgent {
			t.Errorf("missing browser User-Agent, got %q", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	body, err := New().GetBytes(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("GetBytes: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q", body)
	}
}

func TestGetBytesRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := New().GetBytes(context.Background(), srv.URL)
	var rl *cliutil.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected *cliutil.RateLimitError, got %v", err)
	}
	if rl.RetryAfter == 0 {
		t.Error("expected Retry-After to be parsed")
	}
}

func TestGetBytesNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := New().GetBytes(context.Background(), srv.URL); err == nil {
		t.Error("expected error on HTTP 500")
	}
}

// TestNewHasNonZeroTimeout is the acceptance-criteria test for issue #9:
// "unit test: client has non-zero Timeout".
func TestNewHasNonZeroTimeout(t *testing.T) {
	c := New()
	if c.hc.Timeout <= 0 {
		t.Fatalf("New().hc.Timeout = %v, want > 0", c.hc.Timeout)
	}
}

func TestCheckRedirectAllowsAllowlistedHost(t *testing.T) {
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "www.clickthecity.com"}}
	if err := checkRedirect(req, nil); err != nil {
		t.Errorf("checkRedirect for allowlisted host = %v, want nil", err)
	}
}

// TestCheckRedirectAllowsAllowlistedHostCaseInsensitive covers the
// code-review nit that hostnames are case-insensitive (RFC 4343): a
// redirect to an upper-case variant of an allowlisted host must not be
// spuriously blocked.
func TestCheckRedirectAllowsAllowlistedHostCaseInsensitive(t *testing.T) {
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "WWW.POPCORN.APP"}}
	if err := checkRedirect(req, nil); err != nil {
		t.Errorf("checkRedirect for upper-case allowlisted host = %v, want nil", err)
	}
}

func TestCheckRedirectBlocksNonAllowlistedHost(t *testing.T) {
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "evil.example.com"}}
	err := checkRedirect(req, nil)
	if err == nil {
		t.Fatal("checkRedirect for non-allowlisted host = nil, want error")
	}
	if !strings.Contains(err.Error(), "disallowed") {
		t.Errorf("checkRedirect error = %q, want it to mention the disallowed host", err.Error())
	}
}

func TestCheckRedirectStopsAfterMaxRedirects(t *testing.T) {
	via := make([]*http.Request, maxRedirects)
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "www.clickthecity.com"}}
	err := checkRedirect(req, via)
	if err == nil {
		t.Fatal("checkRedirect at redirect cap = nil, want error")
	}
}

// TestGetBytesFollowsRedirectToOffAllowlistHostFails is the acceptance-
// criteria test for issue #9: "redirect off-allowlist fails". Both test
// servers listen on 127.0.0.1:<port>, which is never in
// allowedRedirectHosts, so this deterministically exercises the block
// branch through the real Client.Do path without touching the network or
// requiring control over a real allowlisted domain.
func TestGetBytesFollowsRedirectToOffAllowlistHostFails(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("should never be reached"))
	}))
	defer target.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer srv.Close()

	_, err := New().GetBytes(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected GetBytes to fail on redirect to an off-allowlist host, got nil error")
	}
}

// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

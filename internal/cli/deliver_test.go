// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeliverWebhookRejectsLoopbackByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv(deliverAllowHostsEnv, "")
	err := deliverWebhook(srv.URL, []byte(`{"ok":true}`), false)
	if err == nil {
		t.Fatal("expected loopback webhook to be rejected, got nil error")
	}
	if !strings.Contains(err.Error(), "refusing") {
		t.Errorf("expected a fail-closed refusal error, got: %v", err)
	}
}

func TestDeliverWebhookSucceedsWhenAllowlisted(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		gotBody = buf.Bytes()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv(deliverAllowHostsEnv, "127.0.0.1,::1")
	if err := deliverWebhook(srv.URL, []byte(`{"ok":true}`), false); err != nil {
		t.Fatalf("expected allowlisted webhook to succeed, got: %v", err)
	}
	if string(gotBody) != `{"ok":true}` {
		t.Errorf("server did not receive expected body, got: %q", gotBody)
	}
}

func TestDeliverWebhookWarnsOnPlainHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	if !strings.HasPrefix(srv.URL, "http://") {
		t.Fatalf("httptest.NewServer URL is not http://, got %s", srv.URL)
	}

	t.Setenv(deliverAllowHostsEnv, "127.0.0.1,::1")
	var stderr bytes.Buffer
	old := deliverStderr
	deliverStderr = &stderr
	defer func() { deliverStderr = old }()

	if err := deliverWebhook(srv.URL, []byte(`{}`), false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "plain http") {
		t.Errorf("expected plain-http warning on stderr, got: %q", stderr.String())
	}
}

func TestDeliverWebhookRefusesRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	t.Setenv(deliverAllowHostsEnv, "127.0.0.1,::1")
	err := deliverWebhook(redirector.URL, []byte(`{}`), false)
	if err == nil {
		t.Fatal("expected redirect to be refused, got nil error")
	}
	if !strings.Contains(err.Error(), "redirect") {
		t.Errorf("expected a redirect-refusal error, got: %v", err)
	}
}

// TestDeliverWebhookHTTPSAllowlistedSucceeds hermetically proves the
// https/TLS code path works end-to-end: DialContext resolving-and-dialing
// by IP (instead of letting the transport dial by hostname) does not
// break the TLS handshake or certificate validation, because the
// hostname is still what gets used for the connection's SNI/cert check
// — only the actual TCP dial target changes. This is the closest a
// hermetic test can get to acceptance criterion "public https webhook
// still works" (real public-internet reachability is checked manually
// in Task 7, best-effort, since go test cannot depend on internet
// egress being available in every environment this runs in).
func TestDeliverWebhookHTTPSAllowlistedSucceeds(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	old := deliverTLSConfig
	deliverTLSConfig = &tls.Config{RootCAs: pool}
	defer func() { deliverTLSConfig = old }()

	t.Setenv(deliverAllowHostsEnv, "127.0.0.1,::1")
	if err := deliverWebhook(srv.URL, []byte(`{"ok":true}`), false); err != nil {
		t.Fatalf("expected https webhook with a trusted cert to succeed, got: %v", err)
	}
}

// TestDeliverWebhookHostnameResolutionSucceedsWhenAllowlisted exercises the
// hostname-resolution branch of safeDialContext (resolver.LookupIPAddr +
// the multi-candidate loop), which none of the other tests reach — they
// all target an httptest server by its IP-literal URL, so net.ParseIP(host)
// short-circuits straight to the dial-the-literal path. Rewriting the
// target to "localhost" forces an actual DNS lookup, proving the resolved
// candidate gets validated and dialed correctly when allowlisted.
func TestDeliverWebhookHostnameResolutionSucceedsWhenAllowlisted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	target := strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)
	if target == srv.URL {
		t.Fatalf("test server URL %q did not contain 127.0.0.1 to rewrite to localhost", srv.URL)
	}

	t.Setenv(deliverAllowHostsEnv, "127.0.0.1,::1")
	if err := deliverWebhook(target, []byte(`{"ok":true}`), false); err != nil {
		t.Fatalf("expected allowlisted hostname-resolved webhook to succeed, got: %v", err)
	}
}

// TestDeliverWebhookHostnameResolutionRejectsWhenNotAllowlisted is the
// fail-closed counterpart: the same DNS-resolution path, but without an
// allowlist entry, must still refuse — proving the denylist check applies
// to every resolved candidate, not just literal-IP targets.
func TestDeliverWebhookHostnameResolutionRejectsWhenNotAllowlisted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	target := strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)
	if target == srv.URL {
		t.Fatalf("test server URL %q did not contain 127.0.0.1 to rewrite to localhost", srv.URL)
	}

	t.Setenv(deliverAllowHostsEnv, "")
	err := deliverWebhook(target, []byte(`{"ok":true}`), false)
	if err == nil {
		t.Fatal("expected hostname-resolved loopback webhook to be rejected, got nil error")
	}
	if !strings.Contains(err.Error(), "refusing") {
		t.Errorf("expected a fail-closed refusal error, got: %v", err)
	}
}

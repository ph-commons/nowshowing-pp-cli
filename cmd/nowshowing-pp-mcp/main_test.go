// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

func TestResolveBindAddr(t *testing.T) {
	origLookup := lookupHost
	t.Cleanup(func() { lookupHost = origLookup })

	cases := []struct {
		name        string
		addr        string
		stubLookup  func(host string) ([]string, error)
		wantBind    string
		wantLoop    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "literal loopback IPv4",
			addr:     "127.0.0.1:7777",
			wantBind: "127.0.0.1:7777",
			wantLoop: true,
		},
		{
			name:     "literal loopback IPv6",
			addr:     "[::1]:7777",
			wantBind: "[::1]:7777",
			wantLoop: true,
		},
		{
			name:     "literal IPv4-mapped IPv6 loopback",
			addr:     "[::ffff:127.0.0.1]:7777",
			wantBind: "[::ffff:127.0.0.1]:7777",
			wantLoop: true,
		},
		{
			name:     "literal 0.0.0.0",
			addr:     "0.0.0.0:7777",
			wantBind: "0.0.0.0:7777",
			wantLoop: false,
		},
		{
			name:     "literal IPv6 unspecified",
			addr:     "[::]:7777",
			wantBind: "[::]:7777",
			wantLoop: false,
		},
		{
			name:     "empty host binds all interfaces",
			addr:     ":7777",
			wantBind: ":7777",
			wantLoop: false,
		},
		{
			name: "hostname resolves to loopback -- pinned",
			addr: "localhost:7777",
			stubLookup: func(host string) ([]string, error) {
				if host != "localhost" {
					t.Fatalf("unexpected lookup host %q", host)
				}
				return []string{"127.0.0.1", "::1"}, nil
			},
			wantBind: "127.0.0.1:7777",
			wantLoop: true,
		},
		{
			name: "hostname resolves to non-loopback -- not pinned",
			addr: "internal.example:7777",
			stubLookup: func(host string) ([]string, error) {
				return []string{"10.0.0.5"}, nil
			},
			wantBind: "internal.example:7777",
			wantLoop: false,
		},
		{
			name: "hostname resolves to mixed loopback/non-loopback -- not loopback",
			addr: "mixed.example:7777",
			stubLookup: func(host string) ([]string, error) {
				return []string{"127.0.0.1", "203.0.113.5"}, nil
			},
			wantBind: "mixed.example:7777",
			wantLoop: false,
		},
		{
			name: "hostname resolution returns empty slice with no error -- fail closed, not vacuous loopback",
			addr: "empty.example:7777",
			stubLookup: func(host string) ([]string, error) {
				return []string{}, nil
			},
			wantErr:     true,
			errContains: "no addresses",
		},
		{
			name: "hostname resolution failure -- fail closed",
			addr: "nx.example:7777",
			stubLookup: func(host string) ([]string, error) {
				return nil, errors.New("no such host")
			},
			wantErr:     true,
			errContains: "resolving",
		},
		{
			name:    "malformed addr with no port",
			addr:    "127.0.0.1",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.stubLookup != nil {
				lookupHost = tc.stubLookup
			} else {
				lookupHost = origLookup
			}

			bindAddr, loopback, err := resolveBindAddr(tc.addr)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveBindAddr(%q) = (%q, %v, nil), want error", tc.addr, bindAddr, loopback)
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("resolveBindAddr(%q) error = %q, want substring %q", tc.addr, err.Error(), tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveBindAddr(%q) unexpected error: %v", tc.addr, err)
			}
			if bindAddr != tc.wantBind {
				t.Errorf("resolveBindAddr(%q) bindAddr = %q, want %q", tc.addr, bindAddr, tc.wantBind)
			}
			if loopback != tc.wantLoop {
				t.Errorf("resolveBindAddr(%q) loopback = %v, want %v", tc.addr, loopback, tc.wantLoop)
			}
		})
	}
}

// withStubs saves and restores the lookupHost and startHTTPServer package
// vars around a run() test case, so cases can't leak stub state into each
// other regardless of execution order. Not run with t.Parallel(): the stubs
// are shared package vars, not per-goroutine state.
func withStubs(t *testing.T, lookup func(string) ([]string, error), start func(*server.StreamableHTTPServer, string) error) {
	t.Helper()
	origLookup := lookupHost
	origStart := startHTTPServer
	t.Cleanup(func() {
		lookupHost = origLookup
		startHTTPServer = origStart
	})
	if lookup != nil {
		lookupHost = lookup
	}
	if start != nil {
		startHTTPServer = start
	}
}

func TestRun_HTTPRefusesNonLoopbackWithoutFlag(t *testing.T) {
	withStubs(t, nil, nil) // no HTTP start expected: refusal happens before startHTTPServer

	var buf bytes.Buffer
	code := run([]string{"--transport", "http", "--addr", "0.0.0.0:9999"}, &buf)

	if code != 2 {
		t.Fatalf("run() code = %d, want 2; stderr:\n%s", code, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "refusing to bind") {
		t.Errorf("stderr missing refusal message: %q", out)
	}
	if !strings.Contains(out, "--insecure-bind") {
		t.Errorf("stderr missing --insecure-bind hint: %q", out)
	}
}

func TestRun_HTTPRefusesUnresolvableHostname(t *testing.T) {
	withStubs(t, func(host string) ([]string, error) {
		return nil, errors.New("no such host")
	}, nil)

	var buf bytes.Buffer
	code := run([]string{"--transport", "http", "--addr", "nx.example:9999"}, &buf)

	if code != 2 {
		t.Fatalf("run() code = %d, want 2; stderr:\n%s", code, buf.String())
	}
	if !strings.Contains(buf.String(), "cannot verify") {
		t.Errorf("stderr missing resolution-error message: %q", buf.String())
	}
}

func TestRun_HTTPApprovedHostnamePinsBindAddr(t *testing.T) {
	var captured string
	withStubs(t, func(host string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}, func(srv *server.StreamableHTTPServer, addr string) error {
		captured = addr
		return nil
	})

	var buf bytes.Buffer
	code := run([]string{"--transport", "http", "--addr", "localhost:9999"}, &buf)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr:\n%s", code, buf.String())
	}
	if captured != "127.0.0.1:9999" {
		t.Errorf("startHTTPServer captured addr = %q, want pinned %q (not the literal --addr string)", captured, "127.0.0.1:9999")
	}
	if !strings.Contains(buf.String(), "127.0.0.1:9999") {
		t.Errorf("log line missing pinned address: %q", buf.String())
	}
}

func TestRun_HTTPApprovedLiteralLoopback(t *testing.T) {
	var captured string
	withStubs(t, nil, func(srv *server.StreamableHTTPServer, addr string) error {
		captured = addr
		return nil
	})

	var buf bytes.Buffer
	code := run([]string{"--transport", "http", "--addr", "127.0.0.1:9999"}, &buf)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr:\n%s", code, buf.String())
	}
	if captured != "127.0.0.1:9999" {
		t.Errorf("startHTTPServer captured addr = %q, want %q", captured, "127.0.0.1:9999")
	}
}

func TestRun_HTTPInsecureBindAllowsNonLoopback(t *testing.T) {
	var captured string
	withStubs(t, nil, func(srv *server.StreamableHTTPServer, addr string) error {
		captured = addr
		return nil
	})

	var buf bytes.Buffer
	code := run([]string{"--transport", "http", "--addr", "0.0.0.0:9999", "--insecure-bind"}, &buf)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr:\n%s", code, buf.String())
	}
	if strings.Contains(buf.String(), "refusing to bind") {
		t.Errorf("stderr should not contain a refusal message when --insecure-bind is set: %q", buf.String())
	}
	if captured != "0.0.0.0:9999" {
		t.Errorf("startHTTPServer captured addr = %q, want unpinned %q", captured, "0.0.0.0:9999")
	}
}

// TestRun_HTTPInsecureBindDoesNotDisablePinning proves --insecure-bind does
// not bypass the loopback-pinning logic for an address that would already
// have passed the gate. A plausible, easy-to-write bug is routing
// "if *insecureBind { start with *addr as-is }" ahead of (or instead of)
// the loopback branch -- which would silently reintroduce the TOCTOU class
// (unpinned hostname reaching the HTTP server) specifically for
// loopback-resolving hostnames combined with --insecure-bind.
func TestRun_HTTPInsecureBindDoesNotDisablePinning(t *testing.T) {
	var captured string
	withStubs(t, func(host string) ([]string, error) {
		return []string{"127.0.0.1"}, nil
	}, func(srv *server.StreamableHTTPServer, addr string) error {
		captured = addr
		return nil
	})

	var buf bytes.Buffer
	code := run([]string{"--transport", "http", "--addr", "localhost:9999", "--insecure-bind"}, &buf)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr:\n%s", code, buf.String())
	}
	if captured != "127.0.0.1:9999" {
		t.Errorf("startHTTPServer captured addr = %q, want pinned %q even with --insecure-bind set", captured, "127.0.0.1:9999")
	}
	if !strings.Contains(buf.String(), "127.0.0.1:9999") {
		t.Errorf("log line missing pinned address: %q", buf.String())
	}
}

func TestRun_HTTPBindFailurePropagatesError(t *testing.T) {
	withStubs(t, nil, func(srv *server.StreamableHTTPServer, addr string) error {
		return errors.New("address already in use")
	})

	var buf bytes.Buffer
	code := run([]string{"--transport", "http", "--addr", "127.0.0.1:9999"}, &buf)

	if code != 1 {
		t.Fatalf("run() code = %d, want 1; stderr:\n%s", code, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "MCP server error:") {
		t.Errorf("stderr missing error prefix: %q", out)
	}
	if !strings.Contains(out, "address already in use") {
		t.Errorf("stderr missing underlying error text: %q", out)
	}
}

func TestRun_UnknownTransport(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--transport", "carrier-pigeon"}, &buf)

	if code != 2 {
		t.Fatalf("run() code = %d, want 2; stderr:\n%s", code, buf.String())
	}
	if !strings.Contains(buf.String(), "unknown --transport") {
		t.Errorf("stderr missing unknown-transport message: %q", buf.String())
	}
}

func TestRun_Help(t *testing.T) {
	var buf bytes.Buffer
	code := run([]string{"--help"}, &buf)

	if code != 0 {
		t.Fatalf("run() code = %d, want 0 for --help; stderr:\n%s", code, buf.String())
	}
}

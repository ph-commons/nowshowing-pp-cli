// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

func TestIsDisallowedIP(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		want bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"ipv4-mapped ipv6 loopback", "::ffff:127.0.0.1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"rfc1918 10/8", "10.0.0.5", true},
		{"rfc1918 172.16/12 low", "172.16.0.1", true},
		{"rfc1918 172.16/12 high", "172.31.255.255", true},
		{"rfc1918 192.168/16", "192.168.1.1", true},
		{"link-local metadata", "169.254.169.254", true},
		{"link-local v6", "fe80::1", true},
		{"unique local v6", "fc00::1", true},
		{"cgnat 100.64/10", "100.64.0.1", true},
		{"limited broadcast", "255.255.255.255", true},
		{"link-local multicast", "224.0.0.1", true},
		{"global multicast (ssdp)", "239.255.255.250", true},
		{"public v4", "8.8.8.8", false},
		{"public v6", "2001:4860:4860::8888", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ip := net.ParseIP(c.ip)
			if ip == nil {
				t.Fatalf("net.ParseIP(%q) returned nil", c.ip)
			}
			if got := isDisallowedIP(ip); got != c.want {
				t.Errorf("isDisallowedIP(%s) = %v, want %v", c.ip, got, c.want)
			}
		})
	}
}

func TestIPAllowlisted(t *testing.T) {
	allow := []string{"127.0.0.1", "10.0.0.0/8"}
	cases := []struct {
		name string
		ip   string
		want bool
	}{
		{"exact ip match", "127.0.0.1", true},
		{"cidr match", "10.5.5.5", true},
		{"no match", "169.254.169.254", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ipAllowlisted(net.ParseIP(c.ip), allow)
			if got != c.want {
				t.Errorf("ipAllowlisted(%s) = %v, want %v", c.ip, got, c.want)
			}
		})
	}
	if ipAllowlisted(nil, allow) {
		t.Error("nil ip must never match")
	}
	if ipAllowlisted(net.ParseIP("127.0.0.1"), nil) {
		t.Error("empty allowlist must never match")
	}
}

func TestParseAllowlistRejectsHostnameEntries(t *testing.T) {
	var stderr bytes.Buffer
	old := deliverStderr
	deliverStderr = &stderr
	defer func() { deliverStderr = old }()

	t.Setenv(deliverAllowHostsEnv, "internal.example.com, 10.0.0.0/8 , 127.0.0.1")
	got := parseAllowlist()

	if len(got) != 2 || got[0] != "10.0.0.0/8" || got[1] != "127.0.0.1" {
		t.Errorf("expected only the IP/CIDR entries to survive, got: %v", got)
	}
	if !strings.Contains(stderr.String(), "internal.example.com") || !strings.Contains(stderr.String(), "not an IP") {
		t.Errorf("expected a warning naming the rejected hostname entry, got: %q", stderr.String())
	}
}

func TestValidateResolvedIP(t *testing.T) {
	if err := validateResolvedIP("dns.google", net.ParseIP("8.8.8.8"), nil); err != nil {
		t.Errorf("public IP should be allowed by default, got: %v", err)
	}
	if err := validateResolvedIP("metadata.internal", net.ParseIP("169.254.169.254"), nil); err == nil {
		t.Error("cloud metadata IP must be rejected by default")
	}
	allow := []string{"169.254.169.254"}
	if err := validateResolvedIP("metadata.internal", net.ParseIP("169.254.169.254"), allow); err != nil {
		t.Errorf("explicitly allowlisted IP should pass, got: %v", err)
	}
}

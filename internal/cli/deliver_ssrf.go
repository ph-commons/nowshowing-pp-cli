// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

// Local patch for issue #8 (SSRF/exfil hardening on --deliver webhook).
// See .printing-press-patches/20260811-harden-deliver-webhook-ssrf.md.

package cli

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

// deliverStderr is the sink for --deliver warnings (plain-http notice,
// rejected allowlist entries). Overridden in tests.
var deliverStderr io.Writer = os.Stderr

// deliverAllowHostsEnv names the escape hatch for deliberate internal
// webhook sinks (a local test receiver, an internal collector, etc).
// Value is a comma-separated list of IP literals or CIDR blocks ONLY —
// see the Task 1 design note in the plan for why hostnames are rejected.
// Entries here bypass the denylist below. Empty (default) means no
// bypass — the CLI fails closed.
const deliverAllowHostsEnv = "NOWSHOWING_DELIVER_ALLOW_HOSTS"

// cgnatBlock is RFC 6598 Shared Address Space (100.64.0.0/10), used by
// carrier-grade NAT and increasingly by internal cloud/k8s networks.
// Not covered by net.IP.IsPrivate() (that's RFC 1918 only), so it needs
// an explicit check.
var cgnatBlock = func() *net.IPNet {
	_, block, err := net.ParseCIDR("100.64.0.0/10")
	if err != nil {
		panic(err) // unreachable: literal is a valid CIDR
	}
	return block
}()

// isDisallowedIP reports whether ip must never be reached by the
// --deliver webhook sink: loopback, unspecified, link-local (this
// range covers the 169.254.169.254 cloud metadata address and IPv6
// fe80::/10), private-use ranges (RFC 1918 for v4, ULA fc00::/7 for
// v6), CGNAT (RFC 6598), the limited broadcast address, and multicast
// (webhooks are never legitimately multicast). net.IP's helpers already
// normalize IPv4-mapped IPv6 addresses (e.g. ::ffff:127.0.0.1) via
// To4() before evaluating, so those are covered without extra code.
func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsPrivate() {
		return true
	}
	if ip.Equal(net.IPv4bcast) {
		return true
	}
	return cgnatBlock.Contains(ip)
}

// parseAllowlist reads deliverAllowHostsEnv into a list of IP literals
// and CIDR blocks. Entries that are neither (most commonly a bare
// hostname) are dropped with a warning rather than silently ignored,
// since a hostname entry would look like it works but actually matches
// nothing — ipAllowlisted only ever compares against resolved IPs.
func parseAllowlist() []string {
	raw := os.Getenv(deliverAllowHostsEnv)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	entries := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if net.ParseIP(p) == nil {
			if _, _, err := net.ParseCIDR(p); err != nil {
				fmt.Fprintf(deliverStderr, "warning: %s entry %q is not an IP literal or CIDR block and will be ignored — hostname entries are not supported (DNS can change what a hostname resolves to between allowlisting and connecting); use the IP or CIDR you intend to trust instead\n", deliverAllowHostsEnv, p)
				continue
			}
		}
		entries = append(entries, p)
	}
	return entries
}

// ipAllowlisted reports whether ip matches an explicit IP or CIDR entry
// in allowlist.
func ipAllowlisted(ip net.IP, allowlist []string) bool {
	if ip == nil {
		return false
	}
	for _, entry := range allowlist {
		if entryIP := net.ParseIP(entry); entryIP != nil && entryIP.Equal(ip) {
			return true
		}
		if _, cidr, err := net.ParseCIDR(entry); err == nil && cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// validateResolvedIP applies the denylist+allowlist policy to a single
// resolved IP for host. Returns a descriptive error if the connection
// must be refused.
func validateResolvedIP(host string, ip net.IP, allowlist []string) error {
	if !isDisallowedIP(ip) {
		return nil
	}
	if ipAllowlisted(ip, allowlist) {
		return nil
	}
	return fmt.Errorf(
		"refusing --deliver webhook to %s (%s): loopback/link-local/private/metadata/multicast/CGNAT addresses are blocked by default; set %s to an IP or CIDR to allow this target deliberately",
		host, ip, deliverAllowHostsEnv,
	)
}

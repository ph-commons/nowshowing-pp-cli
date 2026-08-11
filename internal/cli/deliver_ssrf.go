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
var cgnatBlock = mustParseCIDR("100.64.0.0/10")

// nat64Block is the NAT64/DNS64 well-known prefix (RFC 6052). An
// address in this range embeds an IPv4 address in its low 32 bits.
// Unlike cgnatBlock, this prefix is NOT blanket-denied — see
// isDisallowedIP for why: it's a legitimate way for an IPv6-only
// client network to reach a genuine public IPv4 host, so denying the
// whole prefix would break real webhook delivery on such networks.
var nat64Block = mustParseCIDR("64:ff9b::/96")

func mustParseCIDR(s string) *net.IPNet {
	_, block, err := net.ParseCIDR(s)
	if err != nil {
		panic(err) // unreachable: literal is a valid CIDR
	}
	return block
}

// normalizeEmbeddedIPv4 unwraps an IPv4 address embedded in an IPv6
// address so denylist checks see the real underlying address instead
// of an IPv6 notation that slips past them. net.IP.To4() (and every
// isDisallowedIP check that relies on it, transitively, via
// IsLoopback/IsPrivate/IsLinkLocalUnicast) already handles the
// IPv4-mapped form (::ffff:a.b.c.d, marker bytes 0xff,0xff). It does
// NOT handle the older, still-parseable RFC 4291 IPv4-compatible form
// (::a.b.c.d, marker bytes 0x00,0x00) — net.ParseIP("::127.0.0.1")
// succeeds and every To4()-based check on it returns false, so
// ::127.0.0.1, ::169.254.169.254, and ::10.0.0.5 would otherwise sail
// through the denylist under a different notation for the exact same
// address. This closes that gap explicitly.
//
// The all-zero 96-bit prefix this form uses is bit-for-bit identical
// to IPv6's own unspecified (::) and loopback (::1) addresses — those
// two are reserved separately from the IPv4-compatible block by RFC
// 4291, not embedded-IPv4 forms of 0.0.0.0/0.0.0.1, so they're left
// unnormalized and evaluated by net.IP's own IsUnspecified/IsLoopback
// (which already correctly recognize the 16-byte form) instead of
// being reinterpreted here.
func normalizeEmbeddedIPv4(ip net.IP) net.IP {
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	ip16 := ip.To16()
	if ip16 == nil {
		return ip
	}
	for i := 0; i < 12; i++ {
		if ip16[i] != 0 {
			return ip
		}
	}
	if ip.IsUnspecified() || ip.IsLoopback() {
		return ip
	}
	return net.IP(ip16[12:16])
}

// isDisallowedIP reports whether ip must never be reached by the
// --deliver webhook sink: loopback, unspecified, link-local (this
// range covers the 169.254.169.254 cloud metadata address and IPv6
// fe80::/10), private-use ranges (RFC 1918 for v4, ULA fc00::/7 for
// v6), CGNAT (RFC 6598), the limited broadcast address, and multicast
// (webhooks are never legitimately multicast). ip is normalized first
// (normalizeEmbeddedIPv4) so both the standard IPv4-mapped IPv6 form
// and the deprecated IPv4-compatible form evaluate as the same
// underlying address as their dotted-decimal equivalent.
//
// A nat64Block address is handled differently from the other ranges:
// rather than blanket-denying the whole prefix (which would break
// legitimate webhook delivery from IPv6-only/NAT64 client networks
// reaching a genuine public IPv4 host — DNS64 synthesizes exactly this
// form for A-only domains), the embedded IPv4 address is extracted and
// re-checked through this same function. A synthesized address that
// embeds a denylisted target (loopback/private/metadata/etc — still
// reachable via the NAT64 gateway's own local routing) is denied; one
// that embeds an ordinary public IP is allowed, same as if the target
// had resolved to plain IPv4 directly.
func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	ip = normalizeEmbeddedIPv4(ip)
	if ip.IsLoopback() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsPrivate() {
		return true
	}
	if ip.Equal(net.IPv4bcast) {
		return true
	}
	if cgnatBlock.Contains(ip) {
		return true
	}
	if nat64Block.Contains(ip) {
		embedded := net.IP(ip.To16()[12:16])
		return isDisallowedIP(embedded)
	}
	return false
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
// resolved IP for host. ip is normalized once up front (see
// normalizeEmbeddedIPv4) so the denylist and allowlist checks always
// agree on the same underlying address regardless of which IPv6
// notation it arrived in. Returns a descriptive error if the connection
// must be refused.
func validateResolvedIP(host string, ip net.IP, allowlist []string) error {
	ip = normalizeEmbeddedIPv4(ip)
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

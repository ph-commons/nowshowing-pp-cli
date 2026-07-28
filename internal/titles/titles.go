// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

// Package titles normalizes movie titles and showtime strings so the same
// film or screening compares equal across ClickTheCity and popcorn.app, which
// spell titles and times differently. Ported from the ~/src/nowshowing scraper.
package titles

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	studioPrefixes    = []string{"disney:", "marvel studios'", "marvel studios:", "pixar's", "20th century studios'"}
	qualifierSuffixes = []string{" - live action", " (2d)", " (3d)", " (imax)"}
	nonWord           = regexp.MustCompile(`[^\w\s]`)
	multiSpace        = regexp.MustCompile(`\s+`)
	timeNormRe        = regexp.MustCompile(`^0?(\d{1,2}):(\d{2})(AM|PM)$`)
	timeMinRe         = regexp.MustCompile(`^(\d{1,2}):(\d{2})(AM|PM)$`)
	splitDigits       = regexp.MustCompile(`(\d+)`)
)

// Normalize lowercases a title, strips studio prefixes and format qualifiers
// (e.g. "(3D)", "- Live Action"), removes punctuation, and collapses runs of
// whitespace, yielding a stable key for cross-source matching.
func Normalize(title string) string {
	t := strings.TrimSpace(strings.ToLower(title))
	for _, p := range studioPrefixes {
		if strings.HasPrefix(t, p) {
			t = strings.TrimSpace(t[len(p):])
		}
	}
	// Strip repeatedly so stacked qualifiers ("Movie (3D) (IMAX)") fully reduce.
	for stripped := true; stripped; {
		stripped = false
		for _, s := range qualifierSuffixes {
			if strings.HasSuffix(t, s) {
				t = strings.TrimSpace(t[:len(t)-len(s)])
				stripped = true
			}
		}
	}
	t = nonWord.ReplaceAllString(t, "")
	t = strings.TrimSpace(multiSpace.ReplaceAllString(t, " "))
	return t
}

// NormalizeTime canonicalizes a showtime like " 2:20 PM" or "02:20PM" into
// "2:20PM" so equal screenings compare equal. Unparseable input is returned
// uppercased and space-stripped, unchanged otherwise.
func NormalizeTime(t string) string {
	u := strings.ReplaceAll(strings.ToUpper(t), " ", "")
	m := timeNormRe.FindStringSubmatch(u)
	if m == nil {
		return u
	}
	return m[1] + ":" + m[2] + m[3]
}

// MinutesSinceMidnight converts a NormalizeTime'd string (e.g. "11:40AM") to
// minutes past midnight, or -1 when it does not parse.
func MinutesSinceMidnight(t string) int {
	m := timeMinRe.FindStringSubmatch(t)
	if m == nil {
		return -1
	}
	h, _ := strconv.Atoi(m[1])
	mm, _ := strconv.Atoi(m[2])
	switch {
	case m[3] == "PM" && h != 12:
		h += 12
	case m[3] == "AM" && h == 12:
		h = 0
	}
	return h*60 + mm
}

// NaturalLess reports whether a sorts before b using a case-insensitive,
// numeric-aware comparison so "Cinema 2" sorts before "Cinema 10".
func NaturalLess(a, b string) bool {
	at := splitDigits.Split(strings.ToLower(a), -1)
	an := splitDigits.FindAllString(strings.ToLower(a), -1)
	bt := splitDigits.Split(strings.ToLower(b), -1)
	bn := splitDigits.FindAllString(strings.ToLower(b), -1)
	ai, bi := interleave(at, an), interleave(bt, bn)
	for i := 0; i < len(ai) && i < len(bi); i++ {
		x, y := ai[i], bi[i]
		xn, xIsNum := toInt(x)
		yn, yIsNum := toInt(y)
		if xIsNum && yIsNum {
			if xn != yn {
				return xn < yn
			}
			continue
		}
		if x != y {
			return x < y
		}
	}
	return len(ai) < len(bi)
}

func interleave(text, nums []string) []string {
	out := make([]string, 0, len(text)+len(nums))
	for i := 0; i < len(text) || i < len(nums); i++ {
		if i < len(text) && text[i] != "" {
			out = append(out, text[i])
		}
		if i < len(nums) {
			out = append(out, nums[i])
		}
	}
	return out
}

func toInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

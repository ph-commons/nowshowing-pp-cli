// Copyright 2026 Nestor G Pestelos Jr and contributors. Licensed under Apache-2.0. See LICENSE.

package titles

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Disney: Moana 2", "moana 2"},
		{"Marvel Studios' Thunderbolts*", "thunderbolts"},
		{"Superman (3D)", "superman"},
		{"Lilo & Stitch - Live Action", "lilo stitch"},
		{"  A Minecraft   Movie ", "a minecraft movie"},
		{"F1: The Movie", "f1 the movie"},
	}
	for _, c := range cases {
		if got := Normalize(c.in); got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeTime(t *testing.T) {
	cases := []struct{ in, want string }{
		{"2:20 PM", "2:20PM"},
		{"02:20PM", "2:20PM"},
		{" 11:40 am ", "11:40AM"},
		{"garbage", "GARBAGE"},
	}
	for _, c := range cases {
		if got := NormalizeTime(c.in); got != c.want {
			t.Errorf("NormalizeTime(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMinutesSinceMidnight(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"12:00AM", 0},
		{"12:30AM", 30},
		{"1:00AM", 60},
		{"11:40AM", 700},
		{"12:00PM", 720},
		{"2:20PM", 860},
		{"11:59PM", 1439},
		{"nope", -1},
	}
	for _, c := range cases {
		if got := MinutesSinceMidnight(c.in); got != c.want {
			t.Errorf("MinutesSinceMidnight(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestNaturalLess(t *testing.T) {
	if !NaturalLess("Cinema 2", "Cinema 10") {
		t.Error("Cinema 2 should sort before Cinema 10")
	}
	if NaturalLess("Cinema 10", "Cinema 2") {
		t.Error("Cinema 10 should not sort before Cinema 2")
	}
	if !NaturalLess("Cinema 1", "Cinema 1a") {
		t.Error("shorter prefix should sort first")
	}
	if NaturalLess("Director's Club", "Director's Club") {
		t.Error("equal strings should not be Less")
	}
}
